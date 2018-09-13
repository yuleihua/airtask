package module

import "fmt"

type ModuleInfo struct {
	name       string
	version    string
	prevHandle func() error
	mainHandle func() error
	endHandle  func() error
	errHandle  func() error
}

func (m ModuleInfo) String() string {
	return fmt.Sprintf("name:%s,version:%s", m.name, m.version)
}
