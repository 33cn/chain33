// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package strategy

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/33cn/chain33/cmd/tools/tasks"
	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/33cn/chain33/cmd/tools/util"
)

type genDappStrategy struct {
	strategyBasic

	dappName    string
	dappDir     string
	dappProto   string
	packagePath string
}

func (ad *genDappStrategy) Run() error {
	fmt.Println("Begin generate chain33 dapp code.")
	defer fmt.Println("End generate chain33 dapp code.")
	if !ad.initMember() {
		return fmt.Errorf("InitError")
	}

	return ad.runImpl()
}

func (ad *genDappStrategy) checkParamValid() bool {
	return true
}

func (ad *genDappStrategy) initMember() bool {

	dappName, _ := ad.getParam(types.KeyExecutorName)
	outDir, _ := ad.getParam(types.KeyDappOutDir)
	protoPath, _ := ad.getParam(types.KeyProtobufFile)
	//统一转为小写字母
	dappName = strings.ToLower(dappName)

	if strings.Contains(dappName, " ") {
		mlog.Error("InitGenDapp", "Err", "invalid dapp name", "name", dappName)
		return false
	}

	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		mlog.Error("InitGenDapp", "Err", "$GOPATH not exist")
		return false
	}

	// 默认输出到plugin项目的plugin/dapp/目录下
	if outDir == "" {
		outDir = filepath.Join("github.com", "33cn", "plugin", "plugin", "dapp")
	}
	//兼容win 反斜杠路径
	packPath := strings.Replace(filepath.Join(outDir), string(filepath.Separator), "/", -1)
	//绝对路径
	dappRootDir := filepath.Join(goPath, "src", outDir, dappName)
	//check dapp output directory exist
	if util.CheckPathExisted(dappRootDir) {
		mlog.Error("InitGenDapp", "Err", "generate dapp directory exist", "Dir", dappRootDir)
		return false
	}

	if protoPath != "" {
		bExist, _ := util.CheckFileExists(protoPath)
		if !bExist {
			mlog.Error("InitGenDapp", "Err", "specified proto file not exist", "ProtoFile", protoPath)
			return false
		}
	}

	err := os.MkdirAll(dappRootDir, os.ModePerm)

	if err != nil {
		mlog.Error("GenDappDir", "Err", err, "dir", dappRootDir)
		return false
	}

	ad.dappName = dappName
	ad.dappDir = dappRootDir
	ad.dappProto = protoPath
	ad.packagePath = packPath

	return true
}

func (ad *genDappStrategy) runImpl() error {
	var err error
	tashSlice := ad.buildTask()
	for _, task := range tashSlice {

		err = task.Execute()
		if err != nil {
			mlog.Error("GenDappExecTaskFailed.", "error", err, "taskname", task.GetName())
			break
		}
	}
	return err
}

func (ad *genDappStrategy) buildTask() []tasks.Task {
	taskSlice := make([]tasks.Task, 0)
	taskSlice = append(taskSlice,

		&tasks.GenDappCodeTask{
			DappName:    ad.dappName,
			DappDir:     ad.dappDir,
			ProtoFile:   ad.dappProto,
			PackagePath: ad.packagePath,
		},
		&tasks.FormatDappSourceTask{
			OutputFolder: ad.dappDir,
		},
	)
	return taskSlice
}
