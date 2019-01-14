package task

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	gid "github.com/google/uuid"
	"github.com/rjeczalik/notify"
	log "github.com/sirupsen/logrus"

	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/node/plugin"
	"airman.com/airtask/pkg/common"
	"airman.com/airtask/pkg/common/cmd"
	"airman.com/airtask/pkg/types"
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

type Result struct {
	Id        string    `json:"name"`
	BeginTime time.Time `json:"begin_time"`
	EndTime   time.Time `json:"end_time"`
	Err       string    `json:"error"`
	Result    []byte    `json:"result"`
}

// Manager workers.
type Manager struct {
	backend    Backend
	root       string
	cmdRoot    string
	tw         *TimeWheel
	es         *EventSystem
	modules    map[string]*plugin.Module
	isRunning  bool
	queueSize  int
	addTask    chan Task
	deleteTask chan Task
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.RWMutex
}

func NewManager(backend Backend) *Manager {
	return NewManagerWithTimeWheel(backend, DefaultInterval, DefaultSlotNum, MaxChanSize)
}

func NewManagerWithTimeWheel(backend Backend, interval time.Duration, slotNum, size int) *Manager {
	tw := NewTimeWheel(DefaultInterval, DefaultSlotNum)
	ctx, cancel := context.WithCancel(context.Background())
	Manager := &Manager{
		backend:    backend,
		root:       backend.DataDir(),
		tw:         tw,
		es:         NewEventSystem(),
		modules:    make(map[string]*plugin.Module),
		queueSize:  size,
		addTask:    make(chan Task, size),
		deleteTask: make(chan Task, size),
		ctx:        ctx,
		cancel:     cancel,
	}
	return Manager
}

func (m *Manager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	pluginDir := filepath.Join(m.root, DefaultPluginDir)
	if err := os.MkdirAll(pluginDir, 0700); err != nil {
		return err
	}

	shellsDir := filepath.Join(m.root, DefaultCmdDir)
	if err := os.MkdirAll(shellsDir, 0700); err != nil {
		return err
	}
	m.cmdRoot = shellsDir

	if err := m.loadModules(); err != nil {
		return err
	}

	go m.es.eventLoop(m.ctx)
	go m.filesWatcher()
	go m.update()

	m.isRunning = true
	log.Info("task service is running")
	return nil
}

func (m *Manager) loadModules() error {

	fs, err := common.GetFileList(m.root, "so", true)
	if err != nil {
		log.Error("load modules error, path: %s, error:%v", m.root, err)
		return err
	}
	log.Info("files", fs)
	for i := 0; i < len(fs); i++ {
		fileName := filepath.Base(fs[i])
		file := filepath.Join(m.root, fileName)
		if []byte(fileName)[0] == '.' {
			continue
		}
		files := strings.Split(fileName, ".")
		if len(files) > 1 {
			fileName = files[0]
		}

		var id, v string
		id = fileName
		strs := strings.Split(fileName, "@")
		if len(strs) > 1 {
			v = strs[1]
		} else {
			v = DefaultVersion
			id = fileName + "@" + DefaultVersion
		}
		if _, ok := m.modules[id]; !ok {
			m.modules[id] = plugin.NewModule(file, id, v)
		}
		log.Infof("now load file %s: %s: %s", file, id, v)
	}
	log.Infof("load modules %#v\n", m.modules)
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

func (m *Manager) update() {
	for {
		select {
		case t := <-m.addTask:
			m.mu.Lock()
			m.tw.addTask(&t)
			m.mu.Unlock()

		case t := <-m.deleteTask:

			m.mu.Lock()
			m.tw.removeTask(t.id)
			m.mu.Unlock()

		case <-m.tw.ticker.C:
			m.mu.Lock()
			jobs := m.tw.tickTask()
			m.mu.Unlock()

			if len(jobs) > 0 {
				go m.execute(jobs)
			}

		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) filesWatcher() {
	nc := make(chan notify.EventInfo, MaxChanSize)
	if err := notify.Watch(m.root, nc, notify.Create, notify.Remove, notify.Rename); err != nil {
		log.Fatal("notify watch error", "path", m.root, "error", err)
	}
	defer notify.Stop(nc)

	for {
		select {
		case event := <-nc:
			fileName := filepath.Base(event.Path())
			file := filepath.Join(m.root, fileName)
			if []byte(fileName)[0] == '.' {
				continue
			}
			files := strings.Split(fileName, ".")
			if len(files) > 1 {
				fileName = files[0]
			}

			var id, v string
			id = fileName
			strs := strings.Split(fileName, "@")
			if len(strs) > 1 {
				v = strs[1]
			} else {
				v = DefaultVersion
				id = fileName + "@" + DefaultVersion
			}

			log.Infof("event info: %#v", event)
			switch event.Event() {
			case notify.Create:
				m.mu.Lock()
				if _, ok := m.modules[id]; !ok {
					m.modules[id] = plugin.NewModule(file, id, v)
				}
				m.mu.Unlock()

			case notify.Remove:
				m.mu.Lock()
				delete(m.modules, id)
				m.mu.Unlock()
			}
			log.Infof("now event file %s: %s: %s", file, id, v)
			log.Infof("load modules %#v\n", m.modules)

		case <-m.ctx.Done():
			return
		}
	}
}

func (m *Manager) execute(jobs []TaskLite) {
	m.mu.Lock()
	defer m.mu.Unlock()

	r := ExecuteJobs(m.cmdRoot, jobs, m.modules)
	log.Warnf("ExecuteJobs results: %#v\n", r)

	m.es.resultTaskCh <- r
}

func ExecuteJobs(root string, jobs []TaskLite, modules map[string]*plugin.Module) []Result {
	results := make([]Result, len(jobs))
	ctx := context.Background()

	for i := 0; i < len(jobs); i++ {
		if jobs[i].tt == TaskTypeCmd {
			results[i] = callCmdFile(ctx, jobs[i].id, root, jobs[i].id, jobs[i].retryTimes)
		} else {
			if m := modules[jobs[i].id]; m != nil {
				begin := time.Now()
				err := plugin.CallModule(ctx, m, jobs[i].retryTimes)
				end := time.Now()
				results[i] = Result{
					Id:        jobs[i].id,
					BeginTime: begin,
					EndTime:   end,
					Err:       err.Error(),
				}
			}
		}
	}
	return results
}

func callCmdFile(ctx context.Context, tid, shellsDir, file string, retry int) Result {
	cmdFile := filepath.Join(shellsDir, file, ".sh")
	begin := time.Now()
	result, err := cmd.ExecCmd(cmdFile, retry)
	end := time.Now()

	return Result{
		Id:        tid,
		BeginTime: begin,
		EndTime:   end,
		Err:       err.Error(),
		Result:    result,
	}
}

// ListModules lists loaded module.
func (m *Manager) ListModules() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ms := make([]string, 0, len(m.modules))
	for id, _ := range m.modules {
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
func (m *Manager) AddTask(name string, delay time.Duration, extra []byte) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	gid.SetNodeID([]byte("dc10-122"))
	id := gid.New()

	taskName := name
	strs := strings.Split(name, "@")
	if len(strs) == 1 {
		taskName = name + "@" + DefaultVersion
	}

	taskType := TaskTypeUnkown
	if strings.HasPrefix(name, "cmd") {
		taskType = TaskTypeCmd
	} else if strings.HasPrefix(name, "lib") {
		taskType = TaskTypePlugin
	}

	if taskType == TaskTypeUnkown {
		return "", cmn.ErrInvalidTaskName
	}

	if taskType == TaskTypeCmd {
		shellsDir := filepath.Join(m.root, DefaultCmdDir)
		if len(extra) == 0 {
			return "", cmn.ErrInvalidParameter
		}
		cmdFile := filepath.Join(shellsDir, id.String(), ".sh")

		err := ioutil.WriteFile(cmdFile, extra, 0755)
		if err != nil {
			log.Error("write cmd file error, %s: %v", cmdFile, err)
			return "", err
		}
	} else if taskType == TaskTypePlugin {
		if _, ok := m.modules[taskName]; !ok {
			log.Error("plugin name error, %s: %#v", taskName, m.modules)
			return "", cmn.ErrInvalidPluginName
		}
	}
	t := NewTask(delay, id.String(), taskName, taskType, 1)
	m.tw.addTask(t)

	m.es.newTaskCh <- NewTaskUUID(id.String())

	return id.String(), nil
}

// AddTask add delay task.
func (m *Manager) DeleteTask(tid string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, err := gid.Parse(tid); err != nil {
		return err
	}

	isDelete := m.tw.removeTask(tid)
	if isDelete {
		shellsDir := filepath.Join(m.root, DefaultCmdDir)
		cmdFile := filepath.Join(shellsDir, tid)
		return common.RemoveFile(cmdFile)
	}
	return nil
}

// AddTask add delay task.
func (m *Manager) GetTask(tid string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, err := gid.Parse(tid); err != nil {
		return "", err
	}

	t := m.tw.getTask(tid)
	if t == nil || t.name == "" {
		return "", cmn.ErrNoTask
	}
	t.id = tid
	return t.String(), nil
}

// AddTask add delay task.
func (m *Manager) CheckTask(tid string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, err := gid.Parse(tid); err != nil {
		return false, err
	}
	return m.tw.checkTask(tid), nil
}
