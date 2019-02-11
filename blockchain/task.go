// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"errors"
	"sync"
	"time"

	"github.com/33cn/chain33/types"
)

//Task 任务
type Task struct {
	sync.Mutex
	cond     *sync.Cond
	start    int64
	end      int64
	isruning bool
	ticker   *time.Timer
	timeout  time.Duration
	cb       func()
	donelist map[int64]struct{}
}

func newTask(timeout time.Duration) *Task {
	t := &Task{}
	t.timeout = timeout
	t.ticker = time.NewTimer(t.timeout)
	t.cond = sync.NewCond(t)
	go t.tick()
	return t
}

func (t *Task) tick() {
	for {
		t.cond.L.Lock()
		for !t.isruning {
			t.cond.Wait()
		}
		t.cond.L.Unlock()
		_, ok := <-t.ticker.C
		if !ok {
			chainlog.Error("task is done", "timer is stop", t.start)
			continue
		}
		t.Lock()
		if err := t.stop(false); err == nil {
			chainlog.Debug("task is done", "timer is stop", t.start)
		}
		t.Unlock()
	}
}

//InProgress 是否在执行
func (t *Task) InProgress() bool {
	t.Lock()
	defer t.Unlock()
	return t.isruning
}

//TimerReset 计时器重置
func (t *Task) TimerReset(timeout time.Duration) {
	t.TimerStop()
	t.ticker.Reset(timeout)
}

//TimerStop 计时器停止
func (t *Task) TimerStop() {
	if !t.ticker.Stop() {
		select {
		case <-t.ticker.C:
		default:
		}
	}
}

//Start 计时器启动
func (t *Task) Start(start, end int64, cb func()) error {
	t.Lock()
	defer t.Unlock()
	if t.isruning {
		return errors.New("task is running")
	}
	if start > end {
		return types.ErrStartBigThanEnd
	}
	chainlog.Debug("task start:", "start", start, "end", end)
	t.isruning = true
	t.TimerReset(t.timeout)
	t.start = start
	t.end = end
	t.cb = cb
	t.donelist = make(map[int64]struct{})
	t.cond.Signal()
	return nil
}

//Done 任务完成
func (t *Task) Done(height int64) {
	t.Lock()
	defer t.Unlock()
	if !t.isruning {
		return
	}
	if height >= t.start && height <= t.end {
		chainlog.Debug("done", "height", height)
		t.done(height)
		t.TimerReset(t.timeout)
	}
}

func (t *Task) stop(runcb bool) error {
	if !t.isruning {
		return errors.New("not running")
	}
	t.isruning = false
	if t.cb != nil && runcb {
		go t.cb()
	}
	t.TimerStop()
	return nil
}

//Cancel 任务取消
func (t *Task) Cancel() error {
	t.Lock()
	defer t.Unlock()
	chainlog.Warn("----task is cancel----")
	return t.stop(false)
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
			chainlog.Debug("----task is done----")
			err := t.stop(true)
			if err != nil {
				chainlog.Debug("----task stop error ----", "err", err)
			}
		}
	}
	t.donelist[height] = struct{}{}
}
