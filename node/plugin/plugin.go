package plugin

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

// go build -buildmode=plugin -o aplugin.so aplugin.go
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

func executeModule(ctx context.Context, m *Module) error {
	p, err := plugin.Open(m.file)
	if err != nil {
		return err
	}

	// err handle
	errHandle, err := p.Lookup("TaskErr")
	if err != nil {
		return err
	}

	main, err := p.Lookup("TaskMain")
	if err != nil {
		return err
	}

	if err := main.(func(ctx context.Context) error)(ctx); err != nil {
		errHandle.(func(ctx context.Context, err error))(ctx, err)
		return err
	}
	return nil
}

func CallModule(ctx context.Context, module *Module, retryTimes int) error {
	times := retryTimes
	if retryTimes == 0 {
		times = 1
	}

	for i := 0; i < times; i++ {
		if err := executeModule(ctx, module); err != nil {
			log.Error("index: %d execute module: %s error: %v", i, module, err)
		} else {
			break
		}
	}
	return nil
}

//type TypeModuleEvent int
//
//const (
//	TypeModuleEventInit TypeModuleEvent = iota
//	TypeModuleEventAdd
//	TypeModuleEventUpdate
//	TypeModuleEventRemove
//)
//
//type ModuleEvent struct {
//	Id    string
//	Event TypeModuleEvent
//}
//
//func ModuleWatcher(root string, chanEvent chan ModuleEvent) {
//	nc := make(chan notify.EventInfo, 32)
//	if err := notify.Watch(root, nc, notify.Create, notify.Remove, notify.Rename); err != nil {
//		log.Fatal("notify watch error", "path", root, "error", err)
//	}
//	defer notify.Stop(nc)
//
//	for {
//		select {
//		case event := <-nc:
//			fileName := filepath.Base(event.Path())
//			if []byte(fileName)[0] == '.' {
//				continue
//			}
//			files := strings.Split(fileName, ".")
//			if len(files) > 1 {
//				fileName = files[0]
//			}
//
//			chanType := TypeModuleEventInit
//			switch event.Event() {
//			case notify.Create:
//				chanType = TypeModuleEventAdd
//
//			case notify.Remove:
//				chanType = TypeModuleEventRemove
//			}
//			if chanType == TypeModuleEventInit {
//				continue
//			}
//
//			chanEvent <- ModuleEvent{Event: chanType, Id: fileName}
//			log.Info("now event file %d: %s", chanType, fileName)
//		}
//	}
//}
