package task

// Backend interface provides the common API services.
type Backend interface {
	// General API
	DataDir() string
	Config() interface{}
}
