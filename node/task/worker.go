package task

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type Job struct {
	Id         string    `json:"id"`
	FuncName   string    `json:"func_name"`
	Timeout    int       `json:"timeout"`
	RetryTimes int       `json:"retry_times"`
	CreateTime time.Time `json:"create_time"`
	PlanTime   time.Time `json:"plan_time"`
}

type Result struct {
	Id        string    `json:"id"`
	FuncName  string    `json:"func_name"`
	BeginTime time.Time `json:"begin_time"`
	EndTime   time.Time `json:"end_time"`
	Err       error     `json:"err"`
}

type Worker struct {
	mu        sync.Mutex
	ctx       context.Context
	workCh    chan Job
	returnCh  chan<- Result
	isRunning int32 // isRunning indicates whether the agent is currently mining
}

func NewWorker(ctx context.Context, chanSize int) *Worker {
	w := &Worker{
		ctx:    ctx,
		workCh: make(chan Job, chanSize),
	}
	return w
}

func (w *Worker) Work() chan<- Job             { return w.workCh }
func (w *Worker) SetReturnCh(ch chan<- Result) { w.returnCh = ch }

func (w *Worker) Stop() {
	if !atomic.CompareAndSwapInt32(&w.isRunning, 1, 0) {
		return // agent already stopped
	}

done:
	// Empty work channel
	for {
		select {
		case <-w.workCh:
		default:
			break done
		}
	}
}

func (w *Worker) Start() {
	if !atomic.CompareAndSwapInt32(&w.isRunning, 0, 1) {
		return // agent already started
	}
	go w.update()
}

func (w *Worker) update() {
out:
	for {
		select {
		case job := <-w.workCh:
			w.mu.Lock()
			go excuse(&job, w.ctx)
			w.mu.Unlock()
		case <-w.ctx.Done():
			break out
		}
	}
}

func excuse(job *Job, ctx context.Context) {

}
