package types

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

// rpcRequest represents a raw incoming RPC request
type RpcRequest struct {
	Service  string
	Method   string
	Id       interface{}
	IsPubSub bool
	Params   interface{}
	Err      Error // invalid batch element
}

// Backend interface provides the common API services.
type Backend interface {
	Version() string
	Config() interface{}
	Name() string
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
