// Copyright 2018 The huayulei_2003@hotmail.com Authors
// This file is part of the airfk library.
//
// The airfk library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The airfk library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the airfk library. If not, see <http://www.gnu.org/licenses/>.
package task

import (
	"context"
	"path/filepath"

	fsnotify "github.com/rjeczalik/notify"
	log "github.com/sirupsen/logrus"
)

type EventType int

const (
	// AccountCreated
	EventCreated EventType = iota

	// EventRename
	EventRename

	// AccountDropped
	EventDropped
)

type Event struct {
	Type EventType
	File string
}

type Watcher struct {
	ctx       context.Context
	root      string
	chanSize  int
	chanEvent chan Event
}

func (w *Watcher) Start() error {
	// make chan
	chanEvent := make(chan fsnotify.EventInfo, w.chanSize)

	if err := fsnotify.Watch(w.root, chanEvent, fsnotify.Create, fsnotify.Remove, fsnotify.Rename); err != nil {
		log.Error("notify watch error", "path", w.root, "error", err)
		return err
	}

	go func(chanNotify chan fsnotify.EventInfo, chanEvent chan Event) {
		for {
			select {
			case event := <-chanNotify:
				fileName := filepath.Base(event.Path())
				log.Debugf("file event, file:%v, event:%v", fileName, event.Event())

				switch event.Event() {
				case fsnotify.Create:
					// create
					chanEvent <- Event{Type: EventCreated, File: fileName}

				case fsnotify.Rename:
					// rename
					chanEvent <- Event{Type: EventRename, File: fileName}

				case fsnotify.Remove:
					// remove
					chanEvent <- Event{Type: EventDropped, File: fileName}
				}
			case <-w.ctx.Done():
				fsnotify.Stop(chanNotify)
				return
			}
		}
	}(chanEvent, w.chanEvent)

	return nil
}

func (w *Watcher) Event() <-chan Event {
	return w.chanEvent
}
