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

// ----------------------------------------------------------------------------

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

package subscribe

import (
	"context"
	"errors"
	"sync"
	"time"

	"airman.com/airfk/pkg/event"
	"airman.com/airfk/pkg/server"

	cmn "airman.com/airtask/node/common"
)

// Type determines the kind of filter and is used to put the filter in to
// the correct bucket when added.
type Type byte

const (
	// UnknownSubscription indicates an unknown subscription type
	UnknownSubscription Type = iota
	// ResultsTaskSubscription
	ResultsTaskSubscription
	// NewTaskSubscription
	NewTaskSubscription
	// LastSubscription keeps track of the last index
	LastIndexSubscription
)

const (

	// addEvChanSize is the size of channel listening to new task.
	addEvChanSize = 128
	// resultEvChanSize is the size of channel listening to task result.
	resultEvChanSize = 64
)

var (
	ErrInvalidSubscriptionID = errors.New("invalid id")
)

type subscription struct {
	id        server.ID
	typ       Type
	created   time.Time
	results   chan []cmn.Result
	adds      chan int64
	installed chan struct{} // closed when the filter is installed
	err       chan error    // closed when the filter is uninstalled
}

// EventMsg creates subscriptions, processes events and broadcasts them to the
// subscription which match the subscription criteria.
type EventMsg struct {
	backend Backend

	// Subscriptions
	resultsSub event.Subscription // Subscription for result task event
	addsSub    event.Subscription // Subscription for new task event

	// Channels
	install   chan *subscription // install filter for event notification
	uninstall chan *subscription // remove filter for event notification
	resultsCh chan []cmn.Result  // Channel to receive new task result event
	addsCh    chan int64         // Channel to receive new task event
	index     eventIndex

	mu sync.Mutex
}

type eventIndex map[Type]map[server.ID]*subscription

// NewEventSystem creates a new manager that listens for event on the given mux,
// parses and filters them. It uses the all map to retrieve filter changes. The
// work loop holds its own index that is used to forward events to filters.
//
// The returned manager has a loop that needs to be stopped with the Stop function
// or by stopping the given mux.
func NewEventMsg(ctx context.Context, backend Backend) (*EventMsg, error) {
	m := &EventMsg{
		backend:   backend,
		install:   make(chan *subscription),
		uninstall: make(chan *subscription),
		resultsCh: make(chan []cmn.Result, resultEvChanSize),
		addsCh:    make(chan int64, addEvChanSize),
		index:     make(eventIndex),
	}

	// Subscribe events
	m.resultsSub = m.backend.SubscribeResultEvent(m.resultsCh)
	m.addsSub = m.backend.SubscribeNewEvent(m.addsCh)

	// Make sure none of the subscriptions are empty
	if m.resultsSub == nil || m.addsSub == nil {
		return nil, errors.New("subscribe for event system failed")
	}

	for i := UnknownSubscription; i < LastIndexSubscription; i++ {
		m.index[i] = make(map[server.ID]*subscription)
	}

	go m.eventLoop(ctx)
	return m, nil
}

// Subscription is created when the client registers itself for a particular event.
type Subscription struct {
	ID        server.ID
	f         *subscription
	es        *EventMsg
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
			case <-sub.f.results:
			}
		}

		// wait for filter to be uninstalled in work loop before returning
		// this ensures that the manager won't use the event channel which
		// will probably be closed by the client asap after this method returns.
		<-sub.Err()
	})
}

// subscribe installs the subscription in the event broadcast loop.
func (es *EventMsg) subscribe(sub *subscription) *Subscription {
	es.install <- sub
	<-sub.installed
	return &Subscription{ID: sub.id, f: sub, es: es}
}

// SubscribeResultTask creates a subscription that transports result of task.
func (es *EventMsg) SubscribeResultTask(results chan []cmn.Result) *Subscription {
	sub := &subscription{
		id:        server.NewID(),
		typ:       ResultsTaskSubscription,
		created:   time.Now(),
		results:   results,
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

// SubscribeNewTask creates a subscription that transport event of new task.
func (es *EventMsg) SubscribeNewTask(adds chan int64) *Subscription {
	sub := &subscription{
		id:        server.NewID(),
		typ:       NewTaskSubscription,
		created:   time.Now(),
		adds:      adds,
		installed: make(chan struct{}),
		err:       make(chan error),
	}
	return es.subscribe(sub)
}

// broadcast event to filters that match criteria.
func (es *EventMsg) broadcast(ev interface{}) {
	if ev == nil {
		return
	}

	switch e := ev.(type) {
	case []cmn.Result:
		results := make([]cmn.Result, 0, len(e))
		for _, r := range e {
			results = append(results, r)
		}
		for _, f := range es.index[ResultsTaskSubscription] {
			f.results <- results
		}

	case int64:
		for _, f := range es.index[NewTaskSubscription] {
			f.adds <- e
		}
	}
}

// eventLoop (un)installs filters and processes mux events.
func (es *EventMsg) eventLoop(ctx context.Context) {
	for {
		select {
		// Handle subscribed events
		case ev := <-es.resultsCh:
			es.broadcast(ev)

		case ev := <-es.addsCh:
			es.broadcast(ev)

		case f := <-es.install:
			es.mu.Lock()
			es.index[f.typ][f.id] = f
			close(f.installed)
			es.mu.Unlock()

		case f := <-es.uninstall:
			es.mu.Lock()
			delete(es.index[f.typ], f.id)
			close(f.err)
			es.mu.Unlock()

		// System stopped
		case <-es.resultsSub.Err():
			return

		case <-ctx.Done():
			return
		}
	}
}
