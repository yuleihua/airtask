package tw

import (
	"container/list"
	"time"
)

type TimeWheel struct {
	interval   time.Duration
	slots      []*list.List
	timer      map[string]int
	currentPos int // current position
	slotNum    int
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
	}
	// init slots
	for i := 0; i < tw.slotNum; i++ {
		tw.slots[i] = list.New()
	}
	return tw
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

func (tw *TimeWheel) AddTask(task *Task) {
	pos, circle := getPositionAndCircle(task.delay, tw.interval, tw.currentPos, tw.slotNum)
	task.circle = circle
	tw.slots[pos].PushBack(task)
	tw.timer[task.ID] = pos
}

func (tw *TimeWheel) DeleteTask(id string) bool {
	position, ok := tw.timer[id]
	if !ok {
		return true
	}
	isDelete := false
	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.ID == id {
			delete(tw.timer, task.ID)
			l.Remove(e)
			isDelete = true
		}
		e = e.Next()
	}
	return isDelete
}

func (tw *TimeWheel) CheckTask(id string) bool {
	if _, ok := tw.timer[id]; !ok {
		return false
	}
	return true
}

func (tw *TimeWheel) GetTask(id string) *Task {
	position, ok := tw.timer[id]
	if !ok {
		return nil
	}
	var nt = &Task{}
	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.ID == id {
			nt.ID = task.ID
			nt.delay = task.delay
			nt.Type = task.Type
			nt.Name = task.Name
			nt.RetryTimes = task.RetryTimes
			nt.circle = task.circle
			break
		}
		e = e.Next()
	}
	return nt
}

func (tw *TimeWheel) TriggerTask() []TaskLite {
	l := tw.slots[tw.currentPos]
	jobList := tw.scanAllTask(l)

	if tw.currentPos == tw.slotNum-1 {
		tw.currentPos = 0
	} else {
		tw.currentPos++
	}
	return jobList
}

func (tw *TimeWheel) scanAllTask(l *list.List) []TaskLite {
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
			Name:       task.Name,
			ID:         task.ID,
			Type:       task.Type,
			RetryTimes: task.RetryTimes,
		})
		next := e.Next()
		l.Remove(e)
		delete(tw.timer, task.ID)
		e = next
	}
	return jobList
}
