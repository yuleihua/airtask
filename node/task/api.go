package task

import (
	"sync"
)

// PrivateAdminAPI is the collection of administrative API methods exposed only
// over a secure RPC channel.
type PrivateAdminAPI struct {
	backend Backend // Node interfaced by this API
	manager *Manager
	mu      sync.Mutex
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the node itself.
func NewPrivateAdminAPI(backend Backend) *PrivateAdminAPI {
	return &PrivateAdminAPI{backend: backend}
}

type PublicTaskAPI struct {
	backend Backend
	mu      sync.Mutex
}

func NewPublicTaskAPI(backend Backend) *PublicTaskAPI {
	return &PublicTaskAPI{
		backend: backend,
	}
}
