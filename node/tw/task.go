package tw

import (
	"fmt"
	"time"
)

type TaskType byte

const (
	TaskTypeUnkown TaskType = iota
	TaskTypeCmd
	TaskTypeFile
	TaskTypePlugin
)

type Task struct {
	Name       string
	Type       TaskType
	ID         string
	RetryTimes int

	delay  time.Duration // 延迟时间
	circle int           // 时间轮需要转动几圈
}

type TaskLite struct {
	Name       string
	Type       TaskType
	ID         string
	RetryTimes int
}

func NewTask(delay time.Duration, id, name string, t TaskType, retry int) *Task {
	d := int(delay.Seconds())
	if d < 1 {
		d = 1
	}
	r := retry
	if retry == 0 {
		r = 1
	}
	return &Task{
		delay:      time.Duration(d) * time.Second,
		Name:       name,
		ID:         id,
		Type:       t,
		RetryTimes: r,
	}
}

func (t *Task) String() string {
	return fmt.Sprintf("id:%s,name:%s,delay:%v,retry:%d,circle:%d",
		t.ID, t.Name, t.delay, t.RetryTimes, t.circle)
}
