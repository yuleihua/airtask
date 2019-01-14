package task

import (
	"container/list"
	"fmt"
	"time"
)

type TimeWheel struct {
	interval   time.Duration // 一格时间间隔
	ticker     *time.Ticker
	slots      []*list.List // 时间轮槽
	timer      map[string]int
	currentPos int // 当前指针指向哪一个槽
	slotNum    int // 槽数量
}

type TaskType int

const (
	TaskTypeUnkown TaskType = iota
	TaskTypeCmd
	TaskTypeFile
	TaskTypePlugin
)

type Task struct {
	delay      time.Duration // 延迟时间
	circle     int           // 时间轮需要转动几圈
	tt         TaskType
	name       string
	id         string // 定时器唯一标识, 用于删除定时器
	retryTimes int
}

func NewTask(delay time.Duration, id, name string, t TaskType, retry int) *Task {
	d := delay.Seconds()
	if d < 1 {
		d = 1
	}
	r := retry
	if retry == 0 {
		r = 1
	}
	return &Task{
		delay:      delay,
		name:       name,
		id:         id,
		tt:         t,
		retryTimes: r,
	}
}

func (t *Task) String() string {
	return fmt.Sprintf("id:%s,name:%s,delay:%v,retry:%d,circle:%d",
		t.id, t.name, t.delay, t.retryTimes, t.circle)
}

func NewTimeWheel(interval time.Duration, slotNum int) *TimeWheel {
	if interval <= 0 || slotNum <= 0 {
		return nil
	}

	tw := &TimeWheel{
		interval:   interval,
		slots:      make([]*list.List, slotNum),
		timer:      make(map[string]int),
		currentPos: 0,
		slotNum:    slotNum,
		ticker:     time.NewTicker(interval),
	}

	for i := 0; i < tw.slotNum; i++ {
		tw.slots[i] = list.New()
	}
	return tw
}

func (tw *TimeWheel) Timer() *time.Ticker {
	return tw.ticker
}

func (tw *TimeWheel) Interval() time.Duration {
	return tw.interval
}

func getPositionAndCircle(d time.Duration, interval time.Duration, currentPos, slotNum int) (int, int) {
	delaySeconds := int(d.Seconds())
	intervalSeconds := int(interval.Seconds())
	circle := int(delaySeconds / intervalSeconds / slotNum)
	pos := int(currentPos+delaySeconds/intervalSeconds) % slotNum
	return pos, circle
}

func (tw *TimeWheel) addTask(task *Task) {
	pos, circle := getPositionAndCircle(task.delay, tw.interval, tw.currentPos, tw.slotNum)
	task.circle = circle

	tw.slots[pos].PushBack(task)
	tw.timer[task.id] = pos
}

func (tw *TimeWheel) removeTask(key string) bool {
	position, ok := tw.timer[key]
	if !ok {
		return false
	}
	isDelete := false
	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.id == key {
			delete(tw.timer, task.id)
			l.Remove(e)
			isDelete = true
		}
		e = e.Next()
	}
	return isDelete
}

func (tw *TimeWheel) checkTask(key string) bool {
	if _, ok := tw.timer[key]; !ok {
		return false
	}
	return true
}

func (tw *TimeWheel) getTask(key string) *Task {
	position, ok := tw.timer[key]
	if !ok {
		return nil
	}
	var nt = &Task{}
	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.id == key {
			nt.delay = task.delay
			nt.tt = task.tt
			nt.name = task.name
			nt.retryTimes = task.retryTimes
			nt.circle = task.circle
			break
		}
		e = e.Next()
	}
	return nt
}

type TaskLite struct {
	name       string
	tt         TaskType
	id         string
	retryTimes int
}

func (tw *TimeWheel) tickTask() []TaskLite {
	l := tw.slots[tw.currentPos]
	jobList := tw.scanAndRunTask(l)

	if tw.currentPos == tw.slotNum-1 {
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}
	return jobList
}

func (tw *TimeWheel) scanAndRunTask(l *list.List) []TaskLite {
	var jobList []TaskLite
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.circle > 0 {
			task.circle--
			e = e.Next()
			continue
		}
		// run task
		jobList = append(jobList, TaskLite{
			name:       task.name,
			id:         task.id,
			tt:         task.tt,
			retryTimes: task.retryTimes,
		})
		next := e.Next()
		l.Remove(e)
		delete(tw.timer, task.id)
		e = next
	}
	return nil
}
