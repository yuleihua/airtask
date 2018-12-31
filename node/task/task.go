package task

import (
	"context"
	"sync"
	"sync/atomic"
)

// Manager workers.
type Manager struct {
	ctx       context.Context
	workers   map[int]Worker
	isRunning bool
	queueSize int
	mu        sync.Mutex
}

func NewManager(ctx context.Context, qSize int) *Manager {
	Manager := &Manager{
		queueSize: qSize,
		ctx:       ctx,
	}
	//Manager.Register(NewAgent())
	//go Manager.update()
	return Manager
}

func (m *Manager) update() {

}

func (m *Manager) Start() {
	m.mu.Lock()
	defer m.mu.Unlock()

	log.Info("Starting running operation")
	m.worker.start()
	m.worker.commitNewWork()
	m.isRunning = true
}

func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.StoreInt32(&m.running, 0)
	atomic.StoreInt32(&m.shouldStart, 0)

	m.isRunning = false
}

func (m *Manager) Register(agent Agent) {
	if m.Mining() {
		agent.Start()
	}
	m.worker.register(agent)
}

func (m *Manager) Unregister(agent Agent) {
	m.worker.unregister(agent)
}
