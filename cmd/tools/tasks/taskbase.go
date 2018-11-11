// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tasks

import "github.com/33cn/chain33/common/log/log15"

var (
	mlog = log15.New("module", "task")
)

type TaskBase struct {
	NextTask Task
}

func (this *TaskBase) Execute() error {
	return nil
}

func (this *TaskBase) SetNext(t Task) {
	this.NextTask = t
}

func (this *TaskBase) Next() Task {
	return this.NextTask
}
