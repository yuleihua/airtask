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

	log "github.com/sirupsen/logrus"

	"airman.com/airtask/node/admin"
	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/node/conf"
	"airman.com/airtask/pkg/server"
	"airman.com/airtask/pkg/types"
)

var (
	ErrNodeStopped      = errors.New("node not started")
	ErrNodeRunning      = errors.New("node already running")
	ErrServiceUnknown   = errors.New("unknown service")
	ErrDuplicateService = errors.New("duplicate service")
)

type Node struct {
	serviceFuncs []ServiceConstructor         // ts.cmn.Service constructors (in dependency order)
	services     map[reflect.Type]cmn.Service // Currently running services
	isRunning    bool                         // The node is running or not.
	rpcAPIs      []types.API
	conf         *conf.Config
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
func NewNode(config *conf.Config) (*Node, error) {
	confCopy := *config
	config = &confCopy
	if config.DataDir != "" {
		absdatadir, err := filepath.Abs(config.DataDir)
		if err != nil {
			return nil, err
		}
		config.DataDir = absdatadir
	}
	if config.DataDir != "" {
		if err := os.MkdirAll(config.DataDir, 0700); err != nil {
			return nil, err
		}
	}

	return &Node{
		conf:         config,
		serviceFuncs: []ServiceConstructor{},
		httpEndpoint: fmt.Sprintf("%s:%d", config.HTTPHost, config.HTTPPort),
		wsEndpoint:   fmt.Sprintf("%s:%d", config.WSHost, config.WSPort),
		stop:         make(chan struct{}),
	}, nil
}

// isRunning returns node is running or not.
func (n *Node) IsRunning() bool {
	return n.isRunning
}

// Register injects a new ts.cmn.Service into the node's stack. The ts.cmn.Service created by
// the passed constructor must be unique in its type with regard to sibling ones.
func (n *Node) Register(constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.isRunning {
		return ErrNodeRunning
	}

	n.serviceFuncs = append(n.serviceFuncs, constructor)
	return nil
}

// Start create a live P2P node and starts running it.
func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if n.isRunning {
		return ErrNodeRunning
	}

	// Otherwise copy and specialize the P2P configuration
	services := make(map[reflect.Type]cmn.Service)
	for _, constructor := range n.serviceFuncs {
		// Create a new context for the particular ts.cmn.Service
		ctx := &ServiceContext{
			Services: make(map[reflect.Type]cmn.Service),
		}
		for kind, s := range services { // copy needed for threaded access
			ctx.Services[kind] = s
		}
		// Construct and save the ts.cmn.Service
		s, err := constructor(ctx)
		if err != nil {
			return err
		}
		kind := reflect.TypeOf(s)
		if _, exists := services[kind]; exists {
			log.Errorf("duplicate service: %v: %v", kind, s)
			return ErrDuplicateService
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
			log.Errorf("service start error, %v:%v:%v", kind, service, err)
			return err
		}
		// Mark the service started for potential cleanup
		started = append(started, kind)
	}

	// Lastly start the configured RPC interfaces
	if err := n.startRPC(services); err != nil {
		return err
	}

	n.isRunning = true
	// Finish initializing the startup
	n.services = services
	n.stop = make(chan struct{})

	return nil
}

// startRPC is a helper method to start all the various RPC endpoint during node
// startup. It's not meant to be called at any time afterwards as it makes certain
// assumptions about the state of the node.
func (n *Node) startRPC(services map[reflect.Type]cmn.Service) error {
	// Gather all the possible APIs to surface
	apis := n.apis()
	for _, s := range services {
		apis = append(apis, s.APIs()...)
	}

	if err := n.startHTTP(n.conf, apis); err != nil {
		return err
	}
	if err := n.StartWS(n.conf, apis); err != nil {
		n.stopHTTP()
		return err
	}
	// All API endpoints started successfully
	n.rpcAPIs = apis
	return nil
}

// startHTTP initializes and starts the HTTP RPC endpoint.
func (n *Node) startHTTP(c *conf.Config, apis []types.API) error {
	// Short circuit if the WS endpoint isn't being exposed
	endpoint := fmt.Sprintf("%s:%d", c.HTTPHost, c.HTTPPort)

	// Short circuit if the HTTP endpoint isn't being exposed
	if endpoint == "" {
		return nil
	}
	listener, handler, err := server.StartHTTPEndpoint(endpoint, apis, c.HTTPModules, c.HTTPOrigins)
	if err != nil {
		return err
	}
	log.Infof("HTTP endpoint opened url:%s cors :%s", fmt.Sprintf("http://%s", endpoint), strings.Join(c.HTTPOrigins, ","))
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

		log.Infof("HTTP endpoint closed url:%s", fmt.Sprintf("http://%s", n.httpEndpoint))
	}
	if n.httpHandler != nil {
		n.httpHandler.Stop()
		n.httpHandler = nil
	}
}

// StartWS initializes and starts the websocket RPC endpoint.
func (n *Node) StartWS(c *conf.Config, apis []types.API) error {

	// Short circuit if the WS endpoint isn't being exposed
	endpoint := fmt.Sprintf("%s:%d", c.WSHost, c.WSPort)

	listener, handler, err := server.StartWSEndpoint(endpoint, apis, c.WSModules, c.WSOrigins)
	if err != nil {
		return err
	}
	log.Infof("WebSocket endpoint opened url: %s", fmt.Sprintf("ws://%s", listener.Addr()))
	// All listeners booted successfully
	n.wsEndpoint = endpoint
	n.wsListener = listener
	n.wsHandler = handler

	return nil
}

// StopWS terminates the websocket RPC endpoint.
func (n *Node) StopWS() {
	if n.wsListener != nil {
		n.wsListener.Close()
		n.wsListener = nil

		log.Infof("WebSocket endpoint closed url: %s", fmt.Sprintf("ws://%s", n.wsEndpoint))
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

	if !n.isRunning {
		return ErrNodeStopped
	}

	// Terminate the API, services.
	n.StopWS()
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
	n.isRunning = false

	// unblock n.Wait
	close(n.stop)

	if len(failure.Services) > 0 {
		return failure
	}
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

// ts.cmn.Service retrieves a currently running ts.cmn.Service registered of a specific type.
func (n *Node) Service(service interface{}) error {
	n.lock.RLock()
	defer n.lock.RUnlock()

	// Short circuit if the node's not running
	if !n.isRunning {
		return ErrNodeStopped
	}

	// Otherwise try to find the ts.cmn.Service to return
	element := reflect.ValueOf(service).Elem()
	if running, ok := n.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

// DataDir return the current DataDir used by the application.
func (n *Node) DataDir() string {
	return n.conf.DataDir
}

// HTTPEndpoint retrieves the current HTTP endpoint used by the protocol stack.
func (n *Node) HTTPEndpoint() string {
	return n.httpEndpoint
}

// HTTPHandle
func (n *Node) HTTPHandle() *server.Server {
	return n.httpHandler
}

// WSEndpoint retrieves the current WS endpoint used by the protocol stack.
func (n *Node) WSEndpoint() string {
	return n.wsEndpoint
}

// WSHandle
func (n *Node) WSHandle() *server.Server {
	return n.wsHandler
}

// Version return application version.
func (n *Node) Version() string {
	return n.conf.Version.String()
}

// Version return application version.
func (n *Node) NodeId() string {
	return n.conf.GetNodeId()
}

// Name return application name.
func (n *Node) Name() string {
	return n.conf.Name
}

// conf.Config return application configs.
func (n *Node) Config() interface{} {
	return n.conf
}

// RpcAPIs return application apis.
func (n *Node) RpcAPIs() []types.API {
	return n.rpcAPIs
}

// conf.Config return application configs.
func (n *Node) Services() []cmn.Service {
	n.lock.RLock()
	defer n.lock.RUnlock()

	services := make([]cmn.Service, 0, len(n.services))
	for _, s := range n.services {
		services = append(services, s)
	}
	return services
}

// apis returns the collection of RPC descriptors this node offers.
func (n *Node) apis() []types.API {
	return []types.API{
		{
			Namespace: "admin",
			Version:   "1.0",
			Service:   admin.NewPrivateAdminAPI(n),
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   admin.NewPublicAdminAPI(n),
			Public:    true,
		},
	}
}
