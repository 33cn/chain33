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
	"strings"

	"github.com/33cn/chain33/cmd/tools/tasks"
)

type updateInitStrategy struct {
	strategyBasic
	consRootPath    string
	dappRootPath    string
	storeRootPath   string
	cryptoRootPath  string
	mempoolRootPath string
}

func (up *updateInitStrategy) Run() error {
	mlog.Info("Begin run chain33 update init.go.")
	defer mlog.Info("Run chain33 update init.go finish.")
	if err := up.initMember(); err != nil {
		return err
	}
	return up.runImpl()
}

func (up *updateInitStrategy) initMember() error {
	path, err := up.getParam("path")
	packname, _ := up.getParam("packname")
	gopath := os.Getenv("GOPATH")
	if err != nil || path == "" {
		if len(gopath) > 0 {
			path = filepath.Join(gopath, "/src/github.com/33cn/chain33/plugin/")
		}
	}
	if packname == "" {
		packname = strings.Replace(path, gopath+"/src/", "", 1)
	}
	if len(path) == 0 {
		return errors.New("Chain33 Plugin Not Existed")
	}
	up.consRootPath = fmt.Sprintf("%s/consensus/", path)
	up.dappRootPath = fmt.Sprintf("%s/dapp/", path)
	up.storeRootPath = fmt.Sprintf("%s/store/", path)
	up.cryptoRootPath = fmt.Sprintf("%s/crypto/", path)
	up.mempoolRootPath = fmt.Sprintf("%s/mempool/", path)
	mkdir(up.consRootPath)
	mkdir(up.dappRootPath)
	mkdir(up.storeRootPath)
	mkdir(up.cryptoRootPath)
	mkdir(up.mempoolRootPath)
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
	_ "${packname}/consensus/init" //consensus init
	_ "${packname}/crypto/init"    //crypto init
	_ "${packname}/dapp/init"      //dapp init
	_ "${packname}/store/init"     //store init
    _ "${packname}/mempool/init"   //mempool init
)
`)
		data = bytes.Replace(data, []byte("${packname}"), []byte(packname), -1)
		ioutil.WriteFile(path, data, 0666)
	}
}

func (up *updateInitStrategy) runImpl() error {
	var err error
	task := up.buildTask()
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

func (up *updateInitStrategy) buildTask() tasks.Task {
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,
		&tasks.UpdateInitFileTask{
			Folder: up.consRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: up.dappRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: up.storeRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: up.cryptoRootPath,
		},
		&tasks.UpdateInitFileTask{
			Folder: up.mempoolRootPath,
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
