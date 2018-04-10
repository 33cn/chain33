package blockchain

import (
	"math/rand"
	"testing"
	"time"
)

func TestTask(t *testing.T) {
	task := newTask(time.Second)
	if task.InProgress() {
		task.Cancel()
		t.Log("task not start")
		return
	}
	task.Start(1, 10, nil)
	perm := rand.Perm(10)
	for i := 0; i < len(perm); i++ {
		time.Sleep(time.Millisecond * 500)
		task.Done(int64(perm[i]) + 1)
		if i < len(perm)-1 && !task.InProgress() {
			task.Cancel()
			t.Log("task not done, but InProgress is false")
			return
		}
		if i == len(perm)-1 && task.InProgress() {
			task.Cancel()
			t.Log("task is done, but InProgress is true")
			return
		}
	}
}

func TestTasks(t *testing.T) {
	for n := 0; n < 1000; n++ {
		task := newTask(time.Millisecond * 10)
		if task.InProgress() {
			task.Cancel()
			t.Log("task not start")
			return
		}
		task.Start(1, 10, nil)
		perm := rand.Perm(10)
		for i := 0; i < len(perm); i++ {
			time.Sleep(time.Millisecond / 10)
			task.Done(int64(perm[i]) + 1)
			if i < len(perm)-1 && !task.InProgress() {
				task.Cancel()
				t.Log("task not done, but InProgress is false")
				return
			}
			if i == len(perm)-1 && task.InProgress() {
				task.Cancel()
				t.Log("task is done, but InProgress is true")
				return
			}
		}
	}
}
