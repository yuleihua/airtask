package types

import (
	"reflect"
	"sync"

	"gopkg.in/fatih/set.v0"
)

// API describes the set of methods offered over the RPC interface
type API struct {
	Namespace string      // namespace under which the rpc methods of Service are exposed
	Version   string      // api version for DApp's
	Service   interface{} // receiver instance which holds the methods
	Public    bool        // indication if the methods must be considered safe for public use
}

// callback is a method callback which was registered in the server
type Callback struct {
	Rcvr        reflect.Value  // receiver of method
	Method      reflect.Method // callback
	ArgTypes    []reflect.Type // input argument types
	HasCtx      bool           // method's first argument is a context (not included in argTypes)
	ErrPos      int            // err return idx, of -1 when method cannot return error
	IsSubscribe bool           // indication if the callback is a subscription
}

// service represents a registered object
type Service struct {
	Name          string        // name for service
	Typ           reflect.Type  // receiver type
	Callbacks     Callbacks     // registered handlers
	Subscriptions Subscriptions // available subscriptions/notifications
}

// serverRequest is an incoming request
type ServerRequest struct {
	Id            interface{}
	Svcname       string
	Callb         *Callback
	Args          []reflect.Value
	IsUnsubscribe bool
	Err           Error
}

type ServiceRegistry map[string]*Service // collection of services
type Callbacks map[string]*Callback      // collection of RPC callbacks
type Subscriptions map[string]*Callback  // collection of subscription callbacks

// Server represents a RPC server
type Server struct {
	Services ServiceRegistry

	Run      int32
	CodecsMu sync.Mutex
	Codecs   *set.Set
}

// RpcRequest represents a raw incoming RPC request
type RpcRequest struct {
	Service  string
	Method   string
	Id       interface{}
	IsPubSub bool
	Params   interface{}
	Err      Error // invalid batch element
}

// Error wraps RPC errors, which contain an error code in addition to the message.
type Error interface {
	Error() string  // returns the message
	ErrorCode() int // returns the code
}

// ServerCodec implements reading, parsing and writing RPC messages for the server side of
// a RPC session. Implementations must be go-routine safe since the codec can be called in
// multiple go-routines concurrently.
type ServerCodec interface {
	// Read next request
	ReadRequestHeaders() ([]RpcRequest, bool, Error)
	// Parse request argument to the given types
	ParseRequestArguments(argTypes []reflect.Type, params interface{}) ([]reflect.Value, Error)
	// Assemble success response, expects response id and payload
	CreateResponse(id interface{}, reply interface{}) interface{}
	// Assemble error response, expects response id and error
	CreateErrorResponse(id interface{}, err Error) interface{}
	// Assemble error response with extra information about the error through info
	CreateErrorResponseWithInfo(id interface{}, err Error, info interface{}) interface{}
	// Create notification response
	CreateNotification(id, namespace string, event interface{}) interface{}
	// Write msg to client.
	Write(msg interface{}) error
	// Close underlying data stream
	Close()
	// Closed when underlying connection is closed
	Closed() <-chan interface{}
}
