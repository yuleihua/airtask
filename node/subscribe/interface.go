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
	"airman.com/airfk/pkg/event"

	cmn "airman.com/airtask/node/common"
)

// Backend is a "wallet provider" that may contain a batch of accounts they can
// sign transactions with and upon request, do so.
type Backend interface {

	// SubscribeResultEvent registers a subscription of task results.
	SubscribeResultEvent(ch chan<- []cmn.Result) event.Subscription

	// SubscribeNewEvent registers a subscription of task results.
	SubscribeNewEvent(ch chan<- int64) event.Subscription
}
