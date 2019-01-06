package task

import (
	"container/list"
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

type Task struct {
	delay      time.Duration // 延迟时间
	circle     int           // 时间轮需要转动几圈
	name       string
	id         string // 定时器唯一标识, 用于删除定时器
	retryTimes int
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

func (tw *TimeWheel) removeTask(key string) {
	position, ok := tw.timer[key]
	if !ok {
		return
	}
	l := tw.slots[position]
	for e := l.Front(); e != nil; {
		task := e.Value.(*Task)
		if task.id == key {
			delete(tw.timer, task.id)
			l.Remove(e)
		}
		e = e.Next()
	}
}

type TaskLite struct {
	name       string
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
			retryTimes: task.retryTimes,
		})

		next := e.Next()
		l.Remove(e)
		delete(tw.timer, task.id)
		e = next
	}
}
