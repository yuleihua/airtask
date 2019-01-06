package plugin

import (
	"context"
	"fmt"
	"plugin"
	"strings"

	"github.com/fsnotify/fsnotify"
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
	name       string
	version    string
	mainHandle funcHandle
	errHandle  funcErrHandle
}

func NewModule(name, version string) *Module {
	return &Module{
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

func executeModule(ctx context.Context, module string) error {
	p, err := plugin.Open(module)
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

func CallModule(ctx context.Context, module string, retryTimes int) error {
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

type TypeModuleEvent int

const (
	TypeModuleEventInit TypeModuleEvent = iota
	TypeModuleEventAdd
	TypeModuleEventUpdate
	TypeModuleEventRemove
)

type ModuleEvent struct {
	Id    string
	Event TypeModuleEvent
}

func ModuleWatcher(root string, chanEvent chan ModuleEvent) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	err = watcher.Add(root)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case event := <-watcher.Events:
				chanType := TypeModuleEventInit
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)

				} else if event.Op&fsnotify.Create == fsnotify.Create {
					log.Println("create file:", event.Name)
					chanType = TypeModuleEventAdd

				} else if event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("create file:", event.Name)
					chanType = TypeModuleEventRemove
				} else if event.Op&fsnotify.Rename == fsnotify.Rename {
					log.Println("rename file:", event.Name)
				}

				if chanType == TypeModuleEventInit {
					continue
				}

				id := event.Name
				files := strings.Split(event.Name, ".")
				if len(files) > 1 {
					id = files[0]
				}
				chanEvent <- ModuleEvent{id, chanType}

			case err := <-watcher.Errors:
				log.Println("error:", err)
			}
		}
	}()
	return nil
}
