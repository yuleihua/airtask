package tw

import (
	"testing"
	"time"
)

func TestTasks(t *testing.T) {
	id := "9999-11222-122233434-33434"
	tw := NewTimeWheel(1*time.Second, 3600)
	task := NewTask(5*time.Second, id, "cmdls-l", TaskTypeCmd, 1)
	tw.AddTask(task)

	if isExisted := tw.CheckTask(id); !isExisted {
		t.Fatal("no task")
	} else {
		t.Log("task is existed")
	}

	if task := tw.GetTask(id); task != nil {
		t.Logf("id task info: %#v\n", task)
	} else {
		t.Fatal("no task")
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		isOut := false
		select {
		case <-ticker.C:
			if tasks := tw.TriggerTask(); len(tasks) > 0 {
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

	if task := tw.GetTask(id); task != nil {
		t.Fatalf("id task info: %#v\n", task)
	}
}
