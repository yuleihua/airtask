package tw

import (
	"container/list"
	"time"
)

const (
	defaultInterval = 1 * time.Second
	defaultSlot     = 3600
)

type TimeItem struct {
	index  int
	circle int
	id     int64
}

type TimeWheel struct {
	interval    time.Duration
	slots       []*list.List
	mapItems    map[int64]*TimeItem
	currentSlot int // current indexition
	maxSlot     int
}

// NewTimeWheel creates TimeWheel object.
func NewTimeWheel(interval time.Duration, maxSlot int) *TimeWheel {
	var tw *TimeWheel
	if interval <= 0 || maxSlot <= 0 {
		tw = &TimeWheel{
			interval:    defaultInterval,
			slots:       make([]*list.List, maxSlot),
			mapItems:    make(map[int64]*TimeItem),
			currentSlot: 0,
			maxSlot:     defaultSlot,
		}
	} else {
		tw = &TimeWheel{
			interval:    interval,
			slots:       make([]*list.List, maxSlot),
			mapItems:    make(map[int64]*TimeItem),
			currentSlot: 0,
			maxSlot:     maxSlot,
		}
	}
	// init slots
	for i := 0; i < tw.maxSlot; i++ {
		tw.slots[i] = list.New()
	}
	return tw
}

// Interval
func (tw *TimeWheel) Interval() time.Duration {
	return tw.interval
}

// MaxSlot
func (tw *TimeWheel) MaxSlot() int {
	return tw.maxSlot
}

func calcSlotAndCircle(d time.Duration, interval time.Duration, currentSlot, maxSlot int) (int, int) {
	delaySeconds := int(d.Seconds())
	intervalSeconds := int(interval.Seconds())

	circle := int(delaySeconds / intervalSeconds / maxSlot)
	index := int(currentSlot+delaySeconds/intervalSeconds) % maxSlot

	return index, circle
}

// Add item
func (tw *TimeWheel) Add(item Item) (int, int) {
	// calc
	index, circle := calcSlotAndCircle(item.Delay(), tw.interval, tw.currentSlot, tw.maxSlot)
	new := &TimeItem{
		index:  index,
		circle: circle,
		id:     item.ID(),
	}
	tw.mapItems[item.ID()] = new
	tw.slots[index].PushBack(new)
	return index, circle
}

// Delete item by id
func (tw *TimeWheel) Delete(id int64) bool {
	item, ok := tw.mapItems[id]
	if !ok {
		return true
	}
	isDelete := false
	l := tw.slots[item.index]
	for e := l.Front(); e != nil; {
		job := e.Value.(*TimeItem)
		if job.id == id {
			delete(tw.mapItems, id)
			l.Remove(e)
			isDelete = true
		}
		e = e.Next()
	}
	return isDelete
}

// Check item by id.
func (tw *TimeWheel) Check(id int64) bool {
	if _, ok := tw.mapItems[id]; !ok {
		return false
	}
	return true
}

// Get item by id
func (tw *TimeWheel) Get(id int64) (int, int) {
	item, ok := tw.mapItems[id]
	if !ok {
		return 0, 0
	}
	return item.index, item.circle
}

// Trigger return id array.
func (tw *TimeWheel) Trigger() []int64 {
	l := tw.slots[tw.currentSlot]
	jobList := tw.retrieve(l)

	if tw.currentSlot == tw.maxSlot-1 {
		tw.currentSlot = 0
	} else {
		tw.currentSlot++
	}
	return jobList
}

func (tw *TimeWheel) retrieve(l *list.List) []int64 {
	var jobList []int64
	for e := l.Front(); e != nil; {
		job := e.Value.(*TimeItem)
		if job.circle > 0 {
			job.circle--
			e = e.Next()
			continue
		}
		// run job
		jobList = append(jobList, job.id)
		next := e.Next()
		l.Remove(e)
		delete(tw.mapItems, job.id)
		e = next
	}
	return jobList
}
