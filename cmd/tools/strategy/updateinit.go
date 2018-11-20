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

func (this *updateInitStrategy) Run() error {
	mlog.Info("Begin run chain33 update init.go.")
	defer mlog.Info("Run chain33 update init.go finish.")
	if err := this.initMember(); err != nil {
		return err
	}
	return this.runImpl()
}

func (this *updateInitStrategy) initMember() error {
	path, err := this.getParam("path")
	packname, _ := this.getParam("packname")
	if err != nil || path == "" {
		gopath := os.Getenv("GOPATH")
		if len(gopath) > 0 {
			path = filepath.Join(gopath, "/src/github.com/33cn/chain33/plugin/")
		}
	}
	if len(path) == 0 {
		return errors.New("Chain33 Plugin Not Existed")
	}
	this.consRootPath = fmt.Sprintf("%s/consensus/", path)
	this.dappRootPath = fmt.Sprintf("%s/dapp/", path)
	this.storeRootPath = fmt.Sprintf("%s/store/", path)
	this.cryptoRootPath = fmt.Sprintf("%s/crypto/", path)
	mkdir(this.consRootPath)
	mkdir(this.dappRootPath)
	mkdir(this.storeRootPath)
	mkdir(this.cryptoRootPath)
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

func (this *updateInitStrategy) runImpl() error {
	var err error
	task := this.buildTask()
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

func (this *updateInitStrategy) buildTask() tasks.Task {
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		&tasks.UpdateInitFileTask{
			Folder: this.consRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: this.dappRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: this.storeRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: this.cryptoRootPath,
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
