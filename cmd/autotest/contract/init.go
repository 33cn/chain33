// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contract

import (
	"sync"

	"github.com/33cn/chain33/cmd/autotest/testcase"
	"github.com/BurntSushi/toml"
)

type TestInitConfig struct {
	SimpleCaseArr   []testcase.SimpleCase   `toml:"SimpleCase,omitempty"`
	TransferCaseArr []testcase.TransferCase `toml:"TransferCase,omitempty"`
}

func (caseConf *TestInitConfig) RunTest(caseFile string, wg *sync.WaitGroup) {

	fLog := fileLog.New("module", "Init")
	tLog := stdLog.New("module", "Init")
	if _, err := toml.DecodeFile(caseFile, &caseConf); err != nil {

		tLog.Error("ErrTomlDecode", "Error", err.Error())
		return
	}

	tester := testcase.NewTestOperator(fLog, tLog)

	go tester.AddCaseArray(caseConf.SimpleCaseArr, caseConf.TransferCaseArr)
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()
	tester.WaitTest()
}
