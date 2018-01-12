package blockchain

import (
	"sync"
	"time"
)

type Task struct {
	sync.Mutex
	start    int64
	end      int64
	isruning bool
	ticker   *time.Timer
	timeout  time.Duration
	donelist map[int64]struct{}
}

func newTask(timeout time.Duration) *Task {
	t := &Task{}
	t.timeout = timeout
	t.ticker = time.NewTimer(t.timeout)
	go t.tick()
	return t
}

func (t *Task) tick() {
	for {
		t.Lock()
		isrun := t.isruning
		t.Unlock()
		if !isrun {
			time.Sleep(time.Millisecond)
			continue
		}
		_, ok := <-t.ticker.C
		if !ok {
			chainlog.Error("task is done", "timer is stop", t.start)
			continue
		}
		t.Lock()
		if !t.isruning {
			t.Unlock()
			continue
		}
		chainlog.Error("task is timeout", "task id timeout = ", t.start)
		t.isruning = false
		t.Unlock()
	}
}

func (t *Task) InProgress() bool {
	t.Lock()
	defer t.Unlock()
	return t.isruning
}

func (t *Task) Start(start, end int64) {
	t.Lock()
	defer t.Unlock()
	if t.isruning {
		return
	}
	chainlog.Error("task start:", "start", start, "end", end)
	t.isruning = true
	t.ticker.Reset(t.timeout)
	t.start = start
	t.end = end
	t.donelist = make(map[int64]struct{})
}

func (t *Task) Done(height int64) {
	t.Lock()
	defer t.Unlock()
	if height >= t.start && height <= t.end {
		chainlog.Error("done", "height", height)
		t.done(height)
		t.ticker.Reset(t.timeout)
	}
}

func (t *Task) done(height int64) {
	if height == t.start {
		t.start = t.start + 1
		for i := t.start; i <= t.end; i++ {
			_, ok := t.donelist[i]
			if !ok {
				break
			}
			delete(t.donelist, i)
			t.start = i + 1
			//任务完成
		}
		if t.start > t.end {
			t.isruning = false
			t.ticker.Stop()
		}
	}
	t.donelist[height] = struct{}{}
}
