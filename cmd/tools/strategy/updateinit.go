// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strategy

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/33cn/chain33/cmd/tools/tasks"
)

type updateInitStrategy struct {
	strategyBasic
	consRootPath   string
	dappRootPath   string
	storeRootPath  string
	cryptoRootPath string
}

//Run 执行
func (u *updateInitStrategy) Run() error {
	mlog.Info("Begin run chain33 update init.go.")
	defer mlog.Info("Run chain33 update init.go finish.")
	if err := u.initMember(); err != nil {
		return err
	}
	return u.runImpl()
}

func (u *updateInitStrategy) initMember() error {
	path, err := u.getParam("path")
	packname, _ := u.getParam("packname")
	if err != nil || path == "" {
		gopath := os.Getenv("GOPATH")
		if len(gopath) > 0 {
			path = filepath.Join(gopath, "/src/github.com/33cn/chain33/plugin/")
		}
	}
	if len(path) == 0 {
		return errors.New("Chain33 Plugin Not Existed")
	}
	u.consRootPath = fmt.Sprintf("%s/consensus/", path)
	u.dappRootPath = fmt.Sprintf("%s/dapp/", path)
	u.storeRootPath = fmt.Sprintf("%s/store/", path)
	u.cryptoRootPath = fmt.Sprintf("%s/crypto/", path)
	mkdir(u.consRootPath)
	mkdir(u.dappRootPath)
	mkdir(u.storeRootPath)
	mkdir(u.cryptoRootPath)
	buildInit(path, packname)
	return nil
}

func mkdir(path string) {
	path += "init"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0755)
	}
}

func buildInit(path string, packname string) {
	if packname == "" {
		return
	}
	path += "/init.go"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		var data = []byte(`package plugin

import (
	_ "${packname}/consensus/init"
	_ "${packname}/crypto/init"
	_ "${packname}/dapp/init"
	_ "${packname}/store/init"
)`)
		data = bytes.Replace(data, []byte("${packname}"), []byte(packname), -1)
		ioutil.WriteFile(path, data, 0666)
	}
}

func (u *updateInitStrategy) runImpl() error {
	var err error
	task := u.buildTask()
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

func (u *updateInitStrategy) buildTask() tasks.Task {
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		&tasks.UpdateInitFileTask{
			Folder: u.consRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: u.dappRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: u.storeRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: u.cryptoRootPath,
		},
	)

	task := taskSlice[0]
	sliceLen := len(taskSlice)
	for n := 1; n < sliceLen; n++ {
		task.SetNext(taskSlice[n])
		task = taskSlice[n]
	}
	return taskSlice[0]
}
