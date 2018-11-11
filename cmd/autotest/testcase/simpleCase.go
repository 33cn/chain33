// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testcase

import (
	"strings"
)

//simple case just executes without checking, suitable for init situation

type SimpleCase struct {
	BaseCase
}

type SimplePack struct {
	BaseCasePack
}

func (testCase *SimpleCase) doSendCommand(packID string) (PackFunc, error) {

	result, err := runChain33Cli(strings.Fields(testCase.Command))

	if err != nil {
		return nil, err
	}
	pack := SimplePack{}
	pack.txHash = result
	pack.tCase = testCase

	pack.packID = packID
	pack.checkTimes = 0
	return &pack, nil
}

//simple case needn't check
func (pack *SimplePack) doCheckResult(handlerMap CheckHandlerMap) (bCheck bool, bSuccess bool) {

	bCheck = true
	bSuccess = true
	if strings.Contains(pack.txHash, "Err") || strings.Contains(pack.txHash, "connection refused") {

		bSuccess = false
	}

	return bCheck, bSuccess
}
