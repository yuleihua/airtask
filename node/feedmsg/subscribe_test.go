package feedmsg

import (
	"context"
	"fmt"
	"testing"
	"time"

	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/pkg/event"
)

const msgCycle = 200 * time.Millisecond

type TestBackend struct {
	ctx context.Context

	resultFeed  event.Feed              // Event feed to notify wallet additions/removals
	resultScope event.SubscriptionScope // Subscription scope tracking current live listeners
}

func (t *TestBackend) SubscribeResultEvent(ch chan<- []cmn.Result) event.Subscription {

	// Subscribe the caller and track the subscriber count
	sub := t.resultScope.Track(t.resultFeed.Subscribe(ch))

	// Subscribers require an active notification loop, start it
	go t.updater()

	return sub
}

func (t *TestBackend) updater() {
	for {
		// Wait for an account update or a refresh timeout
		select {
		case <-time.After(msgCycle):
		case <-t.ctx.Done():
			return
		}

		r := cmn.NewResultWithEnd(time.Now().String(), time.Now().Unix(), time.Now().Unix(), "ok", []byte("output is 2046"))
		t.resultFeed.Send([]cmn.Result{*r})
	}
}

func TestEventMsg(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	backend := &TestBackend{ctx: ctx}

	m, err := NewEventMsg(ctx, backend)
	if m == nil || err != nil {
		t.Fatal("no notify")
	}

	go func(ctx context.Context) {
		uuid := make(chan []cmn.Result)
		uuidSub := m.SubscribeResultTask(uuid)

		for {
			select {
			case r := <-uuid:
				fmt.Printf("len(r):%d, %#v\n", len(r), r)
			case <-ctx.Done():
				uuidSub.Unsubscribe()
				return
			}
		}
	}(ctx)

	time.Sleep(11 * time.Second)
}
