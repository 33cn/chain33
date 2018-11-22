// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strategy

import "github.com/33cn/chain33/cmd/tools/tasks"

type simpleCreateExecProjStrategy struct {
	strategyBasic
}

func (s *simpleCreateExecProjStrategy) Run() error {
	mlog.Info("Begin run chain33 simple create dapp project.")
	defer mlog.Info("Run chain33 simple create dapp project finish.")
	if err := s.initMember(); err != nil {
		return err
	}
	return s.runImpl()
}

func (s *simpleCreateExecProjStrategy) runImpl() error {
	// 复制模板目录下的文件到指定的目标目录，同时替换掉文件名
	// 遍历目标文件夹内所有文件，替换内部标签
	// 执行shell命令，生成对应的 pb.go 文件
	// 更新引用文件
	var err error
	task := s.buildTask()
	for {
		if task == nil {
			break
		}
		err = task.Execute()
		if err != nil {
			mlog.Error("Execute command failed.", "error", err, "taskname", task.GetName())
			break
		}
		task = task.Next()
	}
	return err
}

func (s *simpleCreateExecProjStrategy) initMember() error {
	return nil
}

func (s *simpleCreateExecProjStrategy) buildTask() tasks.Task {
	return nil
}
