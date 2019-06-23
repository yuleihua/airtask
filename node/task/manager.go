// Copyright 2018 The huayulei_2003@hotmail.com Authors
// This file is part of the airfk library.
//
// The airfk library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The airfk library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the airfk library. If not, see <http://www.gnu.org/licenses/>.
package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"airman.com/airfk/pkg/common"
	"airman.com/airfk/pkg/common/cmd"
	"airman.com/airfk/pkg/event"
	"airman.com/airfk/pkg/leveldb"
	"airman.com/airfk/pkg/types"
	log "github.com/sirupsen/logrus"
	snowflake "github.com/zheng-ji/goSnowFlake"

	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/node/metrics"
	"airman.com/airtask/node/module"
	"airman.com/airtask/node/store"
	fs "airman.com/airtask/node/subscribe"
	"airman.com/airtask/node/tw"
)

const (
	DefaultInterval = 1 * time.Second
	DefaultSlotNum  = 3600
	MaxChanSize     = 64
	DefaultVersion  = "0.0.1"
)

const (
	DefaultPluginDir = "modules"
	DefaultCmdDir    = "shells"
)

// Manager workers.
type Manager struct {
	backend     Backend
	root        string
	moduleRoot  string
	cmdRoot     string
	genID       *snowflake.IdWorker
	tw          *tw.TimeWheel
	es          *fs.EventMsg
	watchModule *Watcher
	dbTask      *store.Store
	dbResult    *store.Store
	modules     map[string]*module.Module
	isRunning   bool
	queueSize   int
	addTask     chan cmn.Job
	deleteTask  chan int64
	execTask    chan int64

	resultsFeed event.Feed // feed notifying of task result
	scope       event.SubscriptionScope

	addFeed  event.Feed // feed notifying of new task
	addScope event.SubscriptionScope

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

func NewManager(backend Backend) *Manager {
	return NewManagerWithTimeWheel(backend, DefaultInterval, DefaultSlotNum, MaxChanSize)
}

func NewManagerWithTimeWheel(backend Backend, interval time.Duration, slotNum, size int) *Manager {
	twManager := tw.NewTimeWheel(DefaultInterval, DefaultSlotNum)
	ctx, cancel := context.WithCancel(context.Background())
	Manager := &Manager{
		backend:    backend,
		root:       backend.DataDir(),
		tw:         twManager,
		modules:    make(map[string]*module.Module),
		addTask:    make(chan cmn.Job, size),
		deleteTask: make(chan int64, size),
		execTask:   make(chan int64, size),
		queueSize:  size,
		ctx:        ctx,
		cancel:     cancel,
	}
	return Manager
}

// SubscribeResultEvent registers a subscription of task results.
func (m *Manager) SubscribeResultEvent(ch chan<- []cmn.Result) event.Subscription {
	return m.scope.Track(m.resultsFeed.Subscribe(ch))
}

// SubscribeNewEvent registers a subscription of task results.
func (m *Manager) SubscribeNewEvent(ch chan<- int64) event.Subscription {
	return m.addScope.Track(m.addFeed.Subscribe(ch))
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginDir := filepath.Join(m.root, DefaultPluginDir)
	if err := os.MkdirAll(pluginDir, 0700); err != nil {
		return err
	}
	m.moduleRoot = pluginDir

	shellsDir := filepath.Join(m.root, DefaultCmdDir)
	if err := os.MkdirAll(shellsDir, 0700); err != nil {
		return err
	}
	m.cmdRoot = shellsDir

	nodeID, _ := strconv.ParseInt(m.backend.NodeID(), 10, 64)
	genID, err := snowflake.NewIdWorker(nodeID)
	if err != nil {
		return err
	}
	m.genID = genID

	dbTaskFile := filepath.Join(m.root, "task")
	dbTask, err := leveldb.NewLDBDatabase(dbTaskFile, 0, 0)
	if err != nil {
		return err
	}
	m.dbTask = store.NewStore(dbTask, store.TaskPrefix)

	dbResultFile := filepath.Join(m.root, "result")
	dbResult, err := leveldb.NewLDBDatabase(dbResultFile, 0, 0)
	if err != nil {
		return err
	}
	m.dbResult = store.NewStore(dbResult, store.ResultPrefix)

	if err := m.loadModules(); err != nil {
		return err
	}

	fsm, err := fs.NewEventMsg(m.ctx, m)
	if err != nil {
		return err
	}
	m.es = fsm

	if err := m.filesWatcher(); err != nil {
		return err
	}
	go m.update()

	m.isRunning = true
	log.Info("task service is running")
	return nil
}

func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.cancel()
	m.isRunning = false
	log.Info("task service is stopped")

	return nil
}

// apis returns the collection of RPC descriptors this node offers.
func (m *Manager) APIs() []types.API {
	return []types.API{
		{
			Namespace: "task",
			Version:   "1.0",
			Service:   NewPrivateTaskAPI(m),
		}, {
			Namespace: "task",
			Version:   "1.0",
			Service:   NewPublicTaskAPI(m),
			Public:    true,
		},
	}
}

func (m *Manager) loadModules() error {
	// file list
	fs, err := common.GetFileList(m.moduleRoot, "so", true)
	if err != nil {
		log.Errorf("load modules error, path: %s, error:%v", m.root, err)
		return err
	}
	log.Info("files", fs)
	for i := 0; i < len(fs); i++ {
		fileName := filepath.Base(fs[i])
		file := filepath.Join(m.moduleRoot, fileName)
		if []byte(fileName)[0] == '.' {
			continue
		}
		name, version := parseModuleName(fileName)
		id := name + "@" + version
		if _, ok := m.modules[id]; !ok {
			m.modules[id] = module.NewModule(file, id, version)
		}
		log.Infof("now load file %s: %s: %s", file, id, version)
	}
	log.Infof("now all modules %#v\n", m.modules)
	return nil
}

func (m *Manager) update() {
	ticker := time.NewTicker(m.tw.Interval())
	defer ticker.Stop()

	for {
		select {
		case job := <-m.addTask:
			m.addHandle(&job)

		case tid := <-m.deleteTask:
			m.deleteHandle(tid)

		case <-ticker.C:
			m.mu.Lock()
			jobs := m.tw.Trigger()
			m.mu.Unlock()

			if len(jobs) > 0 {
				go m.executeHandle(jobs)
			}

		case ev := <-m.watchModule.Event():
			var fileName string
			file := filepath.Join(m.moduleRoot, fileName)
			fileNames := strings.Split(ev.File, ".")
			if len(fileNames) <= 1 || fileNames[len(fileNames)-1] != "so" {
				continue
			}
			if len(fileNames) > 1 {
				fileName = fileNames[0]
			}

			name, version := parseModuleName(fileName)
			id := name + "@" + version

			switch ev.Type {
			case EventCreated:
				m.mu.Lock()
				if _, ok := m.modules[id]; !ok {
					m.modules[id] = module.NewModule(file, id, version)
				}
				m.mu.Unlock()

			case EventDropped:
				m.mu.Lock()
				delete(m.modules, id)
				m.mu.Unlock()
			}

			log.Infof("event info: %#v", ev)
		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) filesWatcher() error {
	w := &Watcher{ctx: m.ctx, root: m.moduleRoot, chanSize: 128, chanEvent: make(chan Event, 64)}
	if err := w.Start(); err != nil {
		return err
	}
	m.watchModule = w
	return nil
}

func (m *Manager) addHandle(job *cmn.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tw.Add(job)

	return nil
}

func (m *Manager) deleteHandle(tid int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tw.Delete(tid)

	return nil
}

func (m *Manager) executeHandle(jobs []int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Warnf("jobs list: %#v\n", jobs)
	r := m.executeJobs(jobs)
	log.Warnf("executeJobs results: %#v\n", r)

	return nil
}

func (m *Manager) executeJobs(tids []int64) error {
	results := make([]cmn.Result, len(tids))
	ctx := context.Background()

	var job cmn.Job
	for idx, tid := range tids {
		jobBytes, err := m.dbTask.Get(cmn.EncodeItemID(uint64(tid)).Bytes())
		if err != nil {
			results[idx] = cmn.Result{
				ID:        tid,
				BeginTime: time.Now().Unix(),
				EndTime:   time.Now().Unix(),
				ErrorMsg:  err.Error(),
			}
			continue
		}

		if err := json.Unmarshal(jobBytes, &job); err != nil {
			results[idx] = cmn.Result{
				ID:        tid,
				BeginTime: time.Now().Unix(),
				EndTime:   time.Now().Unix(),
				ErrorMsg:  err.Error(),
			}
			continue
		}

		begin := time.Now()
		switch job.Type {
		case cmn.JobTypeCmd:
			output, err := cmd.ExecCmd(string(job.Extra), job.Retry)
			results[idx] = cmn.Result{
				ID:        tid,
				BeginTime: begin.Unix(),
				EndTime:   time.Now().Unix(),
				ErrorMsg:  cmn.ToMsg(err),
				Extra:     output,
			}
		case cmn.JobTypeFile:
			cmdFile := filepath.Join(m.cmdRoot, fmt.Sprintf("%v.sh", tid))
			log.Debugf("cmd file:%s", cmdFile)
			output, err := cmd.ExecCmdFile(cmdFile, job.Retry)
			results[idx] = cmn.Result{
				ID:        tid,
				BeginTime: begin.Unix(),
				EndTime:   time.Now().Unix(),
				ErrorMsg:  cmn.ToMsg(err),
				Extra:     output,
			}

		case cmn.JobTypePlugin:
			if m := m.modules[string(job.Extra)]; m != nil {
				err := m.ExecuteWithRetry(ctx, job.Retry)
				results[idx] = cmn.Result{
					ID:        tid,
					BeginTime: begin.Unix(),
					EndTime:   time.Now().Unix(),
					ErrorMsg:  cmn.ToMsg(err),
				}
			}
		}

		// metric
		metrics.TaskExecuteMeter.Mark(1)
		metrics.TaskExecuteTimer.Update(time.Duration(results[idx].EndTime-results[idx].BeginTime) * time.Second)

		jsonBytes, err := json.Marshal(results[idx])
		if err != nil {
			log.Errorf("json marshal struct of result error, %#v, %v", results[idx], err)
		}
		if err := m.dbResult.Put(cmn.EncodeItemID(uint64(tid)).Bytes(), jsonBytes); err != nil {
			log.Errorf("db put result error, %#v, %v", results[idx], err)
		}
	}

	log.Debugf("results: %#v", results)
	m.resultsFeed.Send(results)
	return nil
}

// ListModules lists loaded module.
func (m *Manager) ListModules() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms := make([]string, 0, len(m.modules))
	for id := range m.modules {
		ms = append(ms, id)
	}
	return ms
}

// ListModules lists loaded module.
func (m *Manager) CheckModule(id string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, ok := m.modules[id]; ok {
		return true, nil
	}
	return false, nil
}

// AddTask add delay task.
func (m *Manager) AddTask(job *cmn.Job) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Debugf("job info %#v, %s", job, string(job.Extra))

	newID, err := m.genID.NextId()
	if err != nil {
		return 0, err
	}
	job.UUID = cmn.EncodeItemID(uint64(newID))

	switch job.Type {
	case cmn.JobTypeCmd:
		log.Debugf("cmd string is %v:%v", job.Name, string(job.Extra))

	case cmn.JobTypeFile:
		if len(job.Extra) == 0 {
			return 0, cmn.ErrInvalidParameter
		}

		cmdFile := filepath.Join(m.cmdRoot, fmt.Sprintf("%v.sh", newID))
		if err := ioutil.WriteFile(cmdFile, job.Extra, 0755); err != nil {
			log.Errorf("write cmd file error, %s: %v", cmdFile, err)
			return 0, err
		}

	case cmn.JobTypePlugin:
		taskName := string(job.Extra)
		versions := strings.Split(string(job.Extra), "@")
		if len(versions) == 1 {
			taskName = string(job.Extra) + "@" + DefaultVersion
		}
		if _, ok := m.modules[taskName]; !ok {
			log.Errorf("plugin name error, %s: %#v", taskName, m.modules)
			return 0, cmn.ErrInvalidPluginName
		}
	}

	jobBytes, err := json.Marshal(job)
	if err != nil {
		return 0, err
	}

	if err := m.dbTask.Put(job.UUID.Bytes(), jobBytes); err != nil {
		return 0, err
	}
	m.tw.Add(job)
	m.addFeed.Send(newID)

	return job.UUID.Int64(), nil
}

// AddTask add delay task.
func (m *Manager) DeleteTask(job *cmn.Job) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Debugf("job info %#v, %s", job, string(job.Extra))

	if err := m.dbTask.Delete(job.UUID.Bytes()); err != nil {
		return err
	}

	isDelete := m.tw.Delete(job.UUID.Int64())
	if isDelete {
		cmdFile := filepath.Join(m.cmdRoot, fmt.Sprintf("%v.sh", job.UUID))
		if common.FileExist(cmdFile) {
			return common.RemoveFile(cmdFile)
		}
	}
	return nil
}

// AddTask add delay task.
func (m *Manager) GetTask(job *cmn.Job) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log.Debugf("job info %#v, %s", job, string(job.Extra))

	jobBytes, err := m.dbTask.Get(job.UUID.Bytes())
	if err != nil {
		return nil, err
	}
	idx, circle := m.tw.Get(job.UUID.Int64())

	return map[string]interface{}{
		"info":   string(jobBytes),
		"index":  idx,
		"circle": circle,
	}, nil
}

// AddTask add delay task.
func (m *Manager) CheckTask(job *cmn.Job) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log.Debugf("job info %#v, %s", job, string(job.Extra))

	if _, err := m.dbTask.Get(job.UUID.Bytes()); err != nil {
		return false, err
	}
	return m.tw.Check(job.UUID.Int64()), nil
}

// AddTask add delay task.
func (m *Manager) GetResult(job *cmn.Job) (map[string]interface{}, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log.Debugf("job info %#v, %s", job, string(job.Extra))

	jobBytes, err := m.dbResult.Get(job.UUID.Bytes())
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"info": string(jobBytes),
	}, nil
}

func parseModuleName(file string) (string, string) {
	fileName := []byte(file)
	var name, version string
	isFound := false
	for idx, c := range fileName {
		if c == '@' {
			name = string(fileName[:idx])
			version = string(fileName[idx+1 : len(fileName)-3])
			isFound = true
		}
	}

	if !isFound {
		name = string(fileName[:len(fileName)-3])
		return name, DefaultVersion
	}

	return name, version
}
