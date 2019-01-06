package task

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"airman.com/airtask/node/plugin"
	log "github.com/sirupsen/logrus"
)

const (
	DefaultInterval = 1 * time.Second
	DefaultSlotNum  = 3600
)

type Result struct {
	Id        string    `json:"name"`
	BeginTime time.Time `json:"begin_time"`
	EndTime   time.Time `json:"end_time"`
	Err       error     `json:"err"`
}

// Manager workers.
type Manager struct {
	tw           *TimeWheel
	root         string
	modules      map[string]*plugin.Module
	isRunning    bool
	queueSize    int
	addTask      chan Task
	deleteTask   chan Task
	changeModule chan plugin.ModuleEvent
	quit         chan struct{}
	mu           sync.Mutex
}

func NewManager(root string, size int) *Manager {
	return NewManagerWithTimeWheel(root, DefaultInterval, DefaultSlotNum, size)
}

func NewManagerWithTimeWheel(root string, interval time.Duration, slotNum, size int) *Manager {
	tw := NewTimeWheel(DefaultInterval, DefaultSlotNum)
	Manager := &Manager{
		root:         root,
		tw:           tw,
		modules:      make(map[string]*plugin.Module),
		queueSize:    size,
		addTask:      make(chan Task, size),
		deleteTask:   make(chan Task, size),
		changeModule: make(chan plugin.ModuleEvent, size),
		quit:         make(chan struct{}),
	}
	return Manager
}

func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := plugin.ModuleWatcher(m.root, m.changeModule); err != nil {
		log.Fatal(err)
	}

	m.isRunning = true
	go m.update()
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	close(m.quit)
	m.isRunning = false
}

func (m *Manager) update() {
	for {
		select {
		case <-m.quit:
			return
		case t := <-m.addTask:
			m.mu.Lock()
			m.tw.addTask(&t)
			m.mu.Unlock()

		case t := <-m.deleteTask:

			m.mu.Lock()
			m.tw.removeTask(t.id)
			m.mu.Unlock()

		case change := <-m.changeModule:

			version := "0.0.1"
			id := change.Id
			strs := strings.Split(change.Id, "@")
			if len(strs) > 1 {
				version = strs[1]
			} else {
				id = change.Id + version
			}

			if change.Event == plugin.TypeModuleEventAdd {
				if _, ok := m.modules[id]; !ok {
					m.modules[id] = plugin.NewModule(id, version)
				}
			} else if change.Event == plugin.TypeModuleEventRemove {
				delete(m.modules, id)
			}
		case <-m.tw.ticker.C:
			m.mu.Lock()
			jobs := m.tw.tickTask()
			m.mu.Unlock()

			go m.execute(jobs)
		}
	}
}

func (m *Manager) execute(jobs []TaskLite) {
	r := ExecuteJobs(m.root, jobs, m.modules)
	log.Warnf("ExecuteJobs results: %#v\n", r)
}

func ExecuteJobs(root string, jobs []TaskLite, modules map[string]*plugin.Module) []Result {
	results := make([]Result, len(jobs))
	ctx := context.Background()

	for i := 0; i < len(jobs); i++ {
		if m := modules[jobs[i].id]; m != nil {
			module := fmt.Sprintf("%s/%s.so", root, jobs[i].id)
			begin := time.Now()
			err := plugin.CallModule(ctx, module, jobs[i].retryTimes)
			end := time.Now()
			results[i] = Result{
				Id:        jobs[i].id,
				BeginTime: begin,
				EndTime:   end,
				Err:       err,
			}
		}
	}
	return results
}
