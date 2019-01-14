package task

import (
	"context"
	"sync"
	"time"

	cmn "airman.com/airtask/node/common"
	"airman.com/airtask/pkg/common/hexutil"
	"airman.com/airtask/pkg/server"
)

// PrivateAdminAPI is the collection of administrative API methods exposed only
// over a secure RPC channel.
type PrivateTaskAPI struct {
	manager *Manager
	mu      sync.Mutex
}

// NewPrivateAdminAPI creates a new API definition for the private admin methods
// of the node itself.
func NewPrivateTaskAPI(manager *Manager) *PrivateTaskAPI {
	return &PrivateTaskAPI{manager: manager}
}

func (api *PublicTaskAPI) AddTask(ctx context.Context, name string, delay int64, extra []byte) (string, error) {
	if name == "" || delay <= 0 {
		return "", cmn.ErrInvalidParameter
	}
	return api.manager.AddTask(name, time.Duration(delay)*time.Second, extra)
}

func (api *PrivateTaskAPI) AddTaskWithDatetime(ctx context.Context, name string, datetime int64, extra hexutil.Bytes) (string, error) {
	if name == "" || datetime <= 0 {
		return "", cmn.ErrInvalidParameter
	}

	dt := time.Unix(datetime, 0)
	now := time.Now()

	if !dt.After(now) {
		return "", cmn.ErrInvalidDatetime
	}
	return api.manager.AddTask(name, dt.Sub(now), extra)
}

func (api *PrivateTaskAPI) AddTaskWithRFC3339(ctx context.Context, name string, datetime string, extra hexutil.Bytes) (string, error) {
	if name == "" || datetime == "" {
		return "", cmn.ErrInvalidParameter
	}

	dt, err := time.Parse(time.RFC3339, datetime)
	if err != nil {
		return "", err
	}

	now := time.Now()
	if !dt.After(now) {
		return "", cmn.ErrInvalidDatetime
	}
	return api.manager.AddTask(name, dt.Sub(now), extra)
}

func (api *PublicTaskAPI) GetTask(ctx context.Context, id string) (string, error) {
	if id == "" {
		return "", cmn.ErrInvalidParameter
	}
	return api.manager.GetTask(id)
}

func (api *PrivateTaskAPI) CheckTask(ctx context.Context, id string) (bool, error) {
	if id == "" {
		return false, cmn.ErrInvalidParameter
	}
	return api.manager.CheckTask(id)
}

func (api *PrivateTaskAPI) DelTask(ctx context.Context, id string) error {
	if id == "" {
		return cmn.ErrInvalidParameter
	}
	return api.manager.DeleteTask(id)
}

func (api *PrivateTaskAPI) GetTaskResult(ctx context.Context) []string {
	return nil
}

type PublicTaskAPI struct {
	manager *Manager
}

func NewPublicTaskAPI(manager *Manager) *PublicTaskAPI {
	return &PublicTaskAPI{
		manager: manager,
	}
}

func (api *PublicTaskAPI) ListModules(ctx context.Context) []string {
	if api.manager != nil {
		return nil
	}
	return api.manager.ListModules()
}

func (api *PublicTaskAPI) CheckModule(ctx context.Context, name string) (bool, error) {
	if api.manager != nil {
		return false, nil
	}
	return api.manager.CheckModule(name)
}

func (api *PublicTaskAPI) Stats(ctx context.Context) []string {
	return nil
}

// NewPendingTransactions creates a subscription that is triggered each time a transaction
// enters the transaction pool and was signed from one of the transactions this nodes manages.
func (api *PublicTaskAPI) NewTaskResults(ctx context.Context) (*server.Subscription, error) {
	notifier, supported := server.NotifierFromContext(ctx)
	if !supported {
		return &server.Subscription{}, server.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		results := make(chan []Result, 128)
		resultsSub := api.manager.es.SubscribeResultTask(results)

		for {
			select {
			case hashes := <-results:
				for _, h := range hashes {
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

// NewHeads send a notification each time a new (header) block is appended to the chain.
func (api *PublicTaskAPI) NewTasks(ctx context.Context) (*server.Subscription, error) {
	notifier, supported := server.NotifierFromContext(ctx)
	if !supported {
		return &server.Subscription{}, server.ErrNotificationsUnsupported
	}

	rpcSub := notifier.CreateSubscription()

	go func() {
		uuid := make(chan TaskUUID)
		uuidSub := api.manager.es.SubscribeNewTask(uuid)

		for {
			select {
			case h := <-uuid:
				notifier.Notify(rpcSub.ID, h)
			case <-rpcSub.Err():
				uuidSub.Unsubscribe()
				return
			case <-notifier.Closed():
				uuidSub.Unsubscribe()
				return
			}
		}
	}()

	return rpcSub, nil
}
