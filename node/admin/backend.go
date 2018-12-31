package admin

import (
	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/node/conf"
	"airman.com/airtask/pkg/server"
	"airman.com/airtask/pkg/types"
)

// Backend interface provides the common API services.
type Backend interface {
	// General Penta API
	IsRunning() bool
	StartWS(c *conf.Config, apis []types.API) error
	StopWS()
	DataDir() string
	WSHandle() *server.Server
	WSEndpoint() string
	Version() string
	Name() string
	Config() interface{}
	RpcAPIs() []types.API
	Services() []cmn.Service
}
