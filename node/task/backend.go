package task

import (
	"sync"

	ts "airman.com/airtask/pkg/types"
)

// Task implements the Task full node service.
type Task struct {
	config  *Config
	manager *Manager
	version string
	name    string
	lock    sync.RWMutex // Protects the variadic fields (e.g. gas price and etherbase)
}

func (s *Task) Manager() *Manager { return s.manager }
func (s *Task) Version() string   { return s.version }
func (s *Task) Config() *Config   { return s.config }
func (s *Task) Name() string      { return s.name }

// APIs return the collection of RPC services the Task package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *Task) APIs() []ts.API {
	var apis []ts.API

	// Append all the local APIs and return
	return append(apis, []ts.API{
		{
			Namespace: "task",
			Version:   "1.0",
			Service:   NewPublicTaskAPI(s),
		},
	}...)
}

func (s *Task) Start() error {
	return nil
}

func (s *Task) Stop() error {
	return nil
}
