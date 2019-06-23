package module

import (
	"context"
	"fmt"
	"plugin"

	log "github.com/sirupsen/logrus"
)

const (
	MainHandleName = "TaskMain"
	ErrHandleName  = "TaskErr"
)

// go build -buildmode=plugin -o plugin@0.0.1.so plugin.go
//
type funcHandle func(ctx context.Context) error
type funcErrHandle func(ctx context.Context, err error)

type Module struct {
	file       string
	name       string
	version    string
	mainHandle funcHandle
	errHandle  funcErrHandle
}

func NewModule(file, name, version string) *Module {
	return &Module{
		file:    file,
		name:    name,
		version: version,
	}
}

func NewModuleWithFuncs(name, version string, main funcHandle, err funcErrHandle) *Module {
	return &Module{
		name:       name,
		version:    version,
		mainHandle: main,
		errHandle:  err,
	}
}

func (m *Module) SetFuncs(main funcHandle, err funcErrHandle) {
	m.mainHandle = main
	m.errHandle = err
}

func (m *Module) String() string {
	return fmt.Sprintf("name:%s,version:%s", m.name, m.version)
}

func (m *Module) Execute(ctx context.Context) error {
	p, err := plugin.Open(m.file)
	if err != nil {
		return err
	}

	// err handle
	errHandle, err := p.Lookup(ErrHandleName)
	if err != nil {
		return err
	}

	main, err := p.Lookup(MainHandleName)
	if err != nil {
		return err
	}

	if err := main.(func(ctx context.Context) error)(ctx); err != nil {
		errHandle.(func(ctx context.Context, err error))(ctx, err)
		return err
	}
	return nil
}

func (m *Module) ExecuteWithRetry(ctx context.Context, retryTimes int) error {
	times := retryTimes
	if retryTimes == 0 {
		times = 1
	}

	for i := 0; i < times; i++ {
		if err := m.Execute(ctx); err != nil {
			log.Error("index: %d execute module: %s error: %v", i, m, err)
		} else {
			break
		}
	}
	return nil
}
