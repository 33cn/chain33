// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package tasks 处理的事务任务
package tasks

// Task 处理的事务任务
type Task interface {
	GetName() string
	Next() Task
	SetNext(t Task)

	Execute() error
}
