package api

import "airman.com/airtask/pkg/event"

// Backend interface provides the common API services.
type Backend interface {
	Version() string
	Config() interface{}
	Name() string

	SubscribeNotifyErrEvent(ch chan<- core.ResultEvent) event.Subscription
	SubscribeNotifySuccEvent(ch chan<- core.ResultEvent) event.Subscription
}

// Service is an individual protocol that can be registered into a server.

type Service interface {

	// APIs retrieves the list of RPC descriptors the service provides
	APIs() []API

	// Start is called after all services have been constructed and the networking
	// layer was also initialized to spawn any goroutines required by the service.
	Start() error

	// Stop terminates all goroutines belonging to the service, blocking until they
	// are all terminated.
	Stop() error
}
