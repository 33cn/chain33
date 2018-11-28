// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types 定义autotest包基础结构体、接口、以及函数
package types

import (
	"errors"
	"strings"
)

//simple case just executes without checking, suitable for init situation

//SimpleCase 简单用例
type SimpleCase struct {
	BaseCase
}

//SimplePack 简单pack
type SimplePack struct {
	BaseCasePack
}

//SendCommand 发送cmd
func (testCase *SimpleCase) SendCommand(packID string) (PackFunc, error) {

	output, err := RunChain33Cli(strings.Fields(testCase.GetCmd()))
	if err != nil {
		return nil, errors.New(output)
	}
	testPack := &SimplePack{}
	pack := testPack.GetBasePack()
	pack.TxHash = output
	pack.TCase = testCase

	pack.PackID = packID
	pack.CheckTimes = 0
	return testPack, nil

}

//CheckResult simple case needn't check
func (pack *SimplePack) CheckResult(handlerMap interface{}) (bCheck bool, bSuccess bool) {

	bCheck = true
	bSuccess = true
	if strings.Contains(pack.TxHash, "Err") || strings.Contains(pack.TxHash, "connection refused") {

		bSuccess = false
	}

	return bCheck, bSuccess
}
