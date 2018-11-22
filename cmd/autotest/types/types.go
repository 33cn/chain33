// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "reflect"

//var
var (
	CliCmd        string                      //chain33 cli可执行文件名
	CheckTimeout  int                         //用例check时超时次数
	autoTestItems = make(map[string]AutoTest) //保存注册的dapp测试类型
)

//AutoTest dapp实现auto test的接口，需要提供每个dapp对应的测试配置类型，并注册
type AutoTest interface {
	GetName() string
	GetTestConfigType() reflect.Type
}

//Init 超时次数等于总超时时间除以每次check时睡眠时间
func Init(cliCmd string, checkTimeout int) {

	CliCmd = cliCmd
	CheckTimeout = checkTimeout
}

//RegisterAutoTest 注册测试配置类型
func RegisterAutoTest(at AutoTest) {

	if at == nil || len(at.GetName()) == 0 {
		return
	}
	dapp := at.GetName()

	if _, ok := autoTestItems[dapp]; ok {
		panic("Register Duplicate Dapp, name = " + dapp)
	}
	autoTestItems[dapp] = at
}

//GetAutoTestConfig 获取测试配置类型
func GetAutoTestConfig(dapp string) reflect.Type {

	if len(dapp) == 0 {

		return nil
	}

	if config, ok := autoTestItems[dapp]; ok {

		return config.GetTestConfigType()
	}

	return nil
}
