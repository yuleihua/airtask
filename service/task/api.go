package task

import (
	"sync"

	"airman.com/airtask/pkg/module"
	ts "airman.com/airtask/pkg/types"
)

type PublicTaskAPI struct {
	backend ts.Backend
	//mux       *event.TypeMux
	quit      chan struct{}
	filtersMu sync.Mutex
	filters   map[string]*module.ModuleInfo
}
