package tw

import (
	"testing"
	"time"
)

type TestItem struct {
	id int64
	d  time.Duration
}

func (t *TestItem) ID() int64 {
	return t.id
}

func (t *TestItem) Delay() time.Duration {
	return t.d
}

func TestTasks(t *testing.T) {
	tw := NewTimeWheel(1*time.Second, 3600)
	task := &TestItem{id: 12, d: 3 * time.Second}
	tw.Add(task)

	if isExisted := tw.Check(task.id); !isExisted {
		t.Fatal("no task")
	} else {
		t.Log("task is existed")
	}

	t1, t2 := tw.Get(task.id)
	t.Logf("id task info: %v:%v\n", t1, t2)

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		isOut := false
		select {
		case <-ticker.C:
			if tasks := tw.Trigger(); len(tasks) > 0 {
				t.Logf("task info: %#v\n", tasks)
				isOut = true
				break
			} else {
				t.Error("no tasks")
			}
		}
		if isOut {
			break
		}
	}

	if isExist := tw.Check(task.id); isExist {
		t.Fatalf("id task info: %#v\n", task)
	}
}
