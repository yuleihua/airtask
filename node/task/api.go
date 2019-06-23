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
	"errors"
	"time"

	"airman.com/airfk/pkg/common/hexutil"
	"airman.com/airfk/pkg/server"
	log "github.com/sirupsen/logrus"

	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/node/metrics"
)

// PrivateAdminAPI is the collection of administrative API methods exposed only
// over a secure RPC channel.
type PrivateTaskAPI struct {
	manager *Manager
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the node itself.
func NewPrivateTaskAPI(manager *Manager) *PrivateTaskAPI {
	return &PrivateTaskAPI{manager: manager}
}

// Job is task job.
type JobArgs struct {
	Name     *string        `json:"name"`
	Extra    *hexutil.Bytes `json:"extra"`
	Type     *string        `json:"type"`
	UUID     uint64         `json:"uuid"`
	Datetime int64          `json:"datetime"`
	Retry    int            `json:"retry"`
	Interval int            `json:"interval"`
}

// toJob convert args to job.
func (args *JobArgs) toJob(isAdd bool) (*cmn.Job, error) {
	log.Debugf("args: %#v", args)

	// check name
	if args.Name == nil {
		return nil, errors.New("no name field")
	}

	if isAdd {
		var jobType cmn.JobType
		if args.Type == nil {
			return nil, errors.New("no type field")
		} else {
			switch *args.Type {
			case "cmd":
				jobType = cmn.JobTypeCmd
			case "sh":
				jobType = cmn.JobTypeFile
			case "plugin":
				jobType = cmn.JobTypePlugin
			default:
				return nil, errors.New("invalid type field")
			}
		}

		retry := args.Retry
		if retry == 0 {
			retry = 1
		}

		interval := args.Interval
		if interval == 0 {
			interval = 1
		}

		if args.Datetime > 0 {
			dt := time.Unix(args.Datetime, 0)
			now := time.Now()

			if !dt.After(now) {
				return nil, cmn.ErrInvalidDatetime
			}
			interval = int(dt.Sub(now).Seconds())
		}

		return &cmn.Job{
			Name:     *args.Name,
			Type:     jobType,
			Retry:    retry,
			Interval: interval,
			AddTime:  time.Now().Unix(),
			Extra:    *args.Extra,
		}, nil
	}

	if args.UUID == 0 {
		return nil, errors.New("invalid uuid field")
	}
	return &cmn.Job{
		UUID: cmn.EncodeItemID(args.UUID),
	}, nil
}

// AddTask adds a task
func (api *PrivateTaskAPI) AddTask(args JobArgs) (int64, error) {
	// metric
	metrics.TaskAddMeter.Mark(1)

	job, err := args.toJob(true)
	if err != nil {
		return 0, err
	}
	return api.manager.AddTask(job)
}

// GetTask get task info
func (api *PrivateTaskAPI) GetTask(args JobArgs) (map[string]interface{}, error) {
	job, err := args.toJob(false)
	if err != nil {
		return nil, err
	}
	return api.manager.GetTask(job)
}

// CheckTask check task is existed or not
func (api *PrivateTaskAPI) CheckTask(args JobArgs) (bool, error) {
	job, err := args.toJob(false)
	if err != nil {
		return false, err
	}
	return api.manager.CheckTask(job)
}

// DeleteTask delete task by id
func (api *PrivateTaskAPI) DeleteTask(args JobArgs) error {
	job, err := args.toJob(false)
	if err != nil {
		return err
	}
	return api.manager.DeleteTask(job)
}

// GetTaskResult get task running result.
func (api *PrivateTaskAPI) GetResult(args JobArgs) (map[string]interface{}, error) {
	job, err := args.toJob(false)
	if err != nil {
		return nil, err
	}
	return api.manager.GetResult(job)
}

// Results creates a subscription that is result of task.
func (api *PrivateTaskAPI) Results(ctx context.Context) (*server.Subscription, error) {
	notifier, supported := server.NotifierFromContext(ctx)
	if !supported {
		return &server.Subscription{}, server.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		results := make(chan []cmn.Result, 128)
		resultsSub := api.manager.es.SubscribeResultTask(results)

		for {
			select {
			case rs := <-results:
				for _, h := range rs {
					notifier.Notify(rpcSub.ID, h)
				}
			case <-rpcSub.Err():
				resultsSub.Unsubscribe()
				return
			case <-notifier.Closed():
				resultsSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

// Results creates a subscription that is result of task.
func (api *PrivateTaskAPI) NewTask(ctx context.Context) (*server.Subscription, error) {
	notifier, supported := server.NotifierFromContext(ctx)
	if !supported {
		return &server.Subscription{}, server.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		tasks := make(chan int64, 128)
		resultsSub := api.manager.es.SubscribeNewTask(tasks)

		for {
			select {
			case t := <-tasks:
				notifier.Notify(rpcSub.ID, t)
			case <-rpcSub.Err():
				resultsSub.Unsubscribe()
				return
			case <-notifier.Closed():
				resultsSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}

type PublicTaskAPI struct {
	manager *Manager
}

// NewPublicTaskAPI creates new PublicTaskAPI.
func NewPublicTaskAPI(manager *Manager) *PublicTaskAPI {
	return &PublicTaskAPI{
		manager: manager,
	}
}

// ListModules list plugins.
func (api *PublicTaskAPI) ListModules(ctx context.Context) []string {
	if api.manager != nil {
		return nil
	}
	return api.manager.ListModules()
}

// CheckModule check plugin
func (api *PublicTaskAPI) CheckModule(name string) (bool, error) {
	if api.manager != nil {
		return false, nil
	}
	return api.manager.CheckModule(name)
}

func (api *PublicTaskAPI) Stats() []string {
	return nil
}
