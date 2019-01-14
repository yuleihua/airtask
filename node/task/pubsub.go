// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package filters implements an ethereum filtering system for block,
// transactions and log events.
package task

import (
	"context"
	"errors"
	"sync"
	"time"

	"airman.com/airtask/pkg/server"
)

// Type determines the kind of filter and is used to put the filter in to
// the correct bucket when added.
type Type byte

const (
	// UnknownSubscription indicates an unknown subscription type
	UnknownSubscription Type = iota
	newTaskSubscription
	resultTaskSubscription
	LastIndexSubscription
)

const (
	newTaskChanSize    = 64
	resultTaskChanSize = 128
	TaskUUIDLength     = 36
)

var (
	ErrInvalidSubscriptionID = errors.New("invalid id")
)

type TaskUUID [TaskUUIDLength]byte

func NewTaskUUID(id string) TaskUUID {
	var tid TaskUUID
	tid.SetBytes([]byte(id))
	return tid
}

func (t *TaskUUID) SetBytes(b []byte) {
	if len(b) > len(t) {
		b = b[len(b)-TaskUUIDLength:]
	}
	copy(t[TaskUUIDLength-len(b):], b)
}

func (t *TaskUUID) Bytes() []byte { return t[:] }

func (t *TaskUUID) String() string { return string(t[:]) }

type subscription struct {
	id          server.ID
	typ         Type
	created     time.Time
	taskIds     chan TaskUUID
	taskResults chan []Result
	installed   chan struct{} // closed when the filter is installed
	err         chan error    // closed when the filter is uninstalled
}

// EventSystem creates subscriptions, processes events and broadcasts them to the
// subscription which match the subscription criteria.
type EventSystem struct {
	// Channels
	install      chan *subscription // install filter for event notification
	uninstall    chan *subscription // remove filter for event notification
	newTaskCh    chan TaskUUID      // Channel to receive new transactions event
	resultTaskCh chan []Result      // Channel to receive new log event
}

// NewEventSystem creates a new manager that listens for event on the given mux,
// parses and filters them. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewEventSystem() *EventSystem {
	m := &EventSystem{
		install:      make(chan *subscription),
		uninstall:    make(chan *subscription),
		newTaskCh:    make(chan TaskUUID, newTaskChanSize),
		resultTaskCh: make(chan []Result, resultTaskChanSize),
	}
	return m
}

// Subscription is created when the client registers itself for a particular event.
type Subscription struct {
	ID        server.ID
	f         *subscription
	es        *EventSystem
	unsubOnce sync.Once
}

// Err returns a channel that is closed when unsubscribed.
func (sub *Subscription) Err() <-chan error {
	return sub.f.err
}

// Unsubscribe uninstalls the subscription from the event broadcast loop.
func (sub *Subscription) Unsubscribe() {
	sub.unsubOnce.Do(func() {
	uninstallLoop:
		for {
			// write uninstall request and consume logs/hashes. This prevents
			// the eventLoop broadcast method to deadlock when writing to the
			// filter event channel while the subscription loop is waiting for
			// this method to return (and thus not reading these events).
			select {
			case sub.es.uninstall <- sub.f:
				break uninstallLoop
			}
		}

		// wait for filter to be uninstalled in work loop before returning
		// this ensures that the manager won't use the event channel which
		// will probably be closed by the client asap after this method returns.
		<-sub.Err()
	})
}

// subscribe installs the subscription in the event broadcast loop.
func (es *EventSystem) subscribe(sub *subscription) *Subscription {
	es.install <- sub
	<-sub.installed
	return &Subscription{ID: sub.id, f: sub, es: es}
}

// SubscribeNewHeads creates a subscription that writes the header of a block that is
// imported in the chain.
func (es *EventSystem) SubscribeNewTask(headers chan TaskUUID) *Subscription {
	sub := &subscription{
		id:        server.NewID(),
		typ:       newTaskSubscription,
		created:   time.Now(),
		taskIds:   make(chan TaskUUID),
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

// SubscribePendingTxs creates a subscription that writes transaction hashes for
// transactions that enter the transaction pool.
func (es *EventSystem) SubscribeResultTask(tasks chan []Result) *Subscription {
	sub := &subscription{
		id:          server.NewID(),
		typ:         resultTaskSubscription,
		created:     time.Now(),
		taskResults: make(chan []Result),
		installed:   make(chan struct{}),
		err:         make(chan error),
	}
	return es.subscribe(sub)
}

type filterIndex map[Type]map[server.ID]*subscription

// broadcast event to filters that match criteria.
func (es *EventSystem) broadcast(filters filterIndex, ev interface{}) {
	if ev == nil {
		return
	}

	switch e := ev.(type) {
	case TaskUUID:
		id := e
		for _, f := range filters[newTaskSubscription] {
			f.taskIds <- id
		}

	case []Result:
		results := make([]Result, 0, len(e))
		for _, r := range e {
			results = append(results, r)
		}
		for _, f := range filters[resultTaskSubscription] {
			f.taskResults <- results
		}
	}
}

// eventLoop (un)installs filters and processes mux events.
func (es *EventSystem) eventLoop(ctx context.Context) {
	index := make(filterIndex)
	for i := UnknownSubscription; i < LastIndexSubscription; i++ {
		index[i] = make(map[server.ID]*subscription)
	}

	for {
		select {
		// Handle subscribed events
		case ev := <-es.newTaskCh:
			es.broadcast(index, ev)
		case ev := <-es.resultTaskCh:
			es.broadcast(index, ev)

		case f := <-es.install:
			index[f.typ][f.id] = f
			close(f.installed)
		case f := <-es.uninstall:
			delete(index[f.typ], f.id)
			close(f.err)

		case <-ctx.Done():
			return
		}
	}
}
