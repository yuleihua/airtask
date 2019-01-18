package feedmsg

import (
	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/pkg/event"
)

// Backend is a "wallet provider" that may contain a batch of accounts they can
// sign transactions with and upon request, do so.
type Backend interface {

	// Subscribe creates an async subscription to receive notifications when the
	// backend detects the arrival or departure of a wallet.
	SubscribeResultEvent(ch chan<- []cmn.Result) event.Subscription
}
