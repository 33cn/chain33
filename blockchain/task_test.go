// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTask(t *testing.T) {
	task := newTask(time.Millisecond * 10)
	if task.InProgress() {
		task.Cancel()
		t.Log("task not start")
		return
	}
	task.Start(1, 10, nil, nil)
	perm := rand.Perm(10)
	for i := 0; i < len(perm); i++ {
		time.Sleep(time.Millisecond * 5)
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
	for n := 0; n < 10; n++ {
		task := newTask(time.Millisecond * 10)
		if task.InProgress() {
			task.Cancel()
			t.Log("task not start")
			return
		}
		task.Start(1, 10, nil, nil)
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

func TestTaskTimeOut(t *testing.T) {
	task := newTask(time.Millisecond * 10)
	if task.InProgress() {
		task.Cancel()
		t.Log("task not start")
		return
	}
	defer task.Cancel()
	timeoutHeight := make(chan int64, 1)

	timeoutcb := func(height int64) {
		fmt.Println("timeout:height", height)
		timeoutHeight <- height
	}
	task.Start(1, 11, nil, timeoutcb)
	perm := rand.Perm(10)
	for i := 0; i < len(perm); i++ {
		task.Done(int64(perm[i]) + 1)
	}
	h := <-timeoutHeight
	assert.Equal(t, h, int64(11))
}
