package node

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	"airman.com/airtask/pkg/server"
	ts "airman.com/airtask/pkg/types"
)

var (
	ErrDatadirUsed    = errors.New("datadir already used by another process")
	ErrNodeStopped    = errors.New("node not started")
	ErrNodeRunning    = errors.New("node already running")
	ErrServiceUnknown = errors.New("unknown service")
	ErrNoNetConfig    = errors.New("no net config")
	ErrNoSuchNetwork  = errors.New("no such network")
)

type Node struct {
	serviceFuncs []ServiceConstructor        // ts.Service constructors (in dependency order)
	services     map[reflect.Type]ts.Service // Currently running services

	rpcAPIs      []ts.API
	conf         *Config
	httpEndpoint string
	httpListener net.Listener   // HTTP RPC listener socket to server API requests
	httpHandler  *server.Server // HTTP RPC request handler to process the API requests
	wsEndpoint   string
	wsListener   net.Listener   // Websocket RPC listener socket to server API requests
	wsHandler    *server.Server // Websocket RPC request handler to process the API requests

	lock sync.RWMutex
	stop chan struct{} // Channel to wait for termination notifications
}

// New creates a new P2P node, ready for protocol registration.
func New(config *Config) (*Node, error) {
	confCopy := *config
	config = &confCopy
	if config.ModuleDir != "" {
		absdatadir, err := filepath.Abs(config.ModuleDir)
		if err != nil {
			return nil, err
		}
		config.ModuleDir = absdatadir
	}
	if config.ModuleDir != "" {
		if err := os.MkdirAll(config.ModuleDir, 0700); err != nil {
			return nil, err
		}
	}

	return &Node{
		conf:         config,
		serviceFuncs: []ServiceConstructor{},
		httpEndpoint: fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort),
		wsEndpoint:   fmt.Sprintf("%s:%d", config.WSHost, config.WSPort),
	}, nil
}

// Register injects a new ts.Service into the node's stack. The ts.Service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

// Start create a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]ts.Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular ts.Service
		ctx := &ServiceContext{
			Services: make(map[reflect.Type]ts.Service),
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.Services[kind] = s
		}
		// Construct and save the ts.Service
		s, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(s)
		if _, exists := services[kind]; exists {
			fmt.Errorf("duplicate service: %v", kind)
		}
		services[kind] = s
	}

	// Start each of the services
	var started []reflect.Type
	for kind, service := range services {
		// Start the next service, stopping all previous upon failure
		if err := service.Start(); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}
			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}

	// Lastly start the configured RPC interfaces
	if err := n.startRPC(services); err != nil {
		return err
	}

	// Finish initializing the startup
	n.services = services
	n.stop = make(chan struct{})

	return nil
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC(services map[reflect.Type]ts.Service) error {
	// Gather all the possible APIs to surface
	apis := n.apis()
	for _, s := range services {
		apis = append(apis, s.APIs()...)
	}

	if err := n.startHTTP(n.httpEndpoint, apis, n.conf.HTTPModules, n.conf.HTTPOrigins); err != nil {
		return err
	}
	if err := n.startWS(n.wsEndpoint, apis, n.conf.WSModules, n.conf.WSOrigins); err != nil {
		n.stopHTTP()
		return err
	}
	// All API endpoints started successfully
	n.rpcAPIs = apis
	return nil
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(endpoint string, apis []ts.API, modules []string, cors []string) error {
	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, handler, err := server.StartHTTPEndpoint(endpoint, apis, modules, cors)
	if err != nil {
		return err
	}
	n.log.Info("HTTP endpoint opened", "url", fmt.Sprintf("http://%s", endpoint), "cors", strings.Join(cors, ","))
	// All listeners booted successfully
	n.httpEndpoint = endpoint
	n.httpListener = listener
	n.httpHandler = handler

	return nil
}

// stopHTTP terminates the HTTP RPC endpoint.
func (n *Node) stopHTTP() {
	if n.httpListener != nil {
		n.httpListener.Close()
		n.httpListener = nil

		log.Info("HTTP endpoint closed", "url", fmt.Sprintf("http://%s", n.httpEndpoint))
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

// startWS initializes and starts the websocket RPC endpoint.
func (n *Node) startWS(endpoint string, apis []ts.API, modules []string, wsOrigins []string) error {
	// Short circuit if the WS endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, handler, err := server.StartWSEndpoint(endpoint, apis, modules, wsOrigins)
	if err != nil {
		return err
	}
	//log.Info("WebSocket endpoint opened", "url", fmt.Sprintf("ws://%s", listener.Addr()))
	// All listeners booted successfully
	n.wsEndpoint = endpoint
	n.wsListener = listener
	n.wsHandler = handler

	return nil
}

// stopWS terminates the websocket RPC endpoint.
func (n *Node) stopWS() {
	if n.wsListener != nil {
		n.wsListener.Close()
		n.wsListener = nil

		//log.Info("WebSocket endpoint closed", "url", fmt.Sprintf("ws://%s", n.wsEndpoint))
	}
	if n.wsHandler != nil {
		n.wsHandler.Stop()
		n.wsHandler = nil
	}
}

// Stop terminates a running node along with all it's services. In the node was
// not started, an error is returned.
func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// Terminate the API, services and the p2p server.
	n.stopWS()
	n.stopHTTP()

	failure := &StopError{
		Services: make(map[reflect.Type]error),
	}
	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	n.rpcAPIs = nil
	n.services = nil

	// unblock n.Wait
	close(n.stop)

	return nil
}

// Restart terminates a running node and boots up a new one in its place. If the
// node isn't running, an error is returned.
func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}
	if err := n.Start(); err != nil {
		return err
	}
	return nil
}

// ts.Service retrieves a currently running ts.Service registered of a specific type.
func (n *Node) Service(service interface{}) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Otherwise try to find the ts.Service to return
	element := reflect.ValueOf(service).Elem()
	if running, ok := n.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

// ModuleDir retrieves the current ModuleDir used by the protocol stack.
// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (n *Node) ModuleDir() string {
	return n.conf.ModuleDir
}

// HTTPEndpoint retrieves the current HTTP endpoint used by the protocol stack.
func (n *Node) HTTPEndpoint() string {
	return n.httpEndpoint
}

// WSEndpoint retrieves the current WS endpoint used by the protocol stack.
func (n *Node) WSEndpoint() string {
	return n.wsEndpoint
}

// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (n *Node) Version() string {
	return n.conf.Version
}

// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (n *Node) Name() string {
	return n.conf.Name
}

// Deprecated: No files should be stored in this directory, use InstanceDir instead.
func (n *Node) Config() interface{} {
	return n.conf
}

// apis returns the collection of RPC descriptors this node offers.
func (n *Node) apis() []ts.API {
	return []ts.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   service.NewPrivateAdminAPI(n),
		}, {
			Namespace: "biz",
			Version:   "1.0",
			Service:   service.NewPublicBizAPI(n),
			Public:    true,
		},
	}
}
