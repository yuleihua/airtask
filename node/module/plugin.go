package module

import (
	"context"
	"fmt"
)

type funcHandle func(ctx context.Context) error
type funcErrHandle func(ctx context.Context, err error)

type Module struct {
	name       string
	version    string
	prevHandle funcHandle
	mainHandle funcHandle
	endHandle  funcHandle
	errHandle  funcErrHandle
}

func NewModule(name, version string, prev, main, end funcHandle, err funcErrHandle) *Module {
	return &Module{
		name:       name,
		version:    version,
		prevHandle: prev,
		mainHandle: main,
		endHandle:  end,
		errHandle:  err,
	}
}

func (m *Module) String() string {
	return fmt.Sprintf("name:%s,version:%s", m.name, m.version)
}
