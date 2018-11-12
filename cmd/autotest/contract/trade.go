// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contract

import (
	"sync"

	"github.com/33cn/chain33/cmd/autotest/testcase"
	"github.com/BurntSushi/toml"
)

type TestTradeConfig struct {
	TokenPreCreateCaseArr    []testcase.TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []testcase.TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []testcase.TransferCase          `toml:"TransferCase,omitempty"`
	SellCaseArr              []testcase.SellCase              `toml:"SellCase,omitempty"`
	DependBuyCaseArr         []testcase.DependBuyCase         `toml:"DependBuyCase,omitempty"`
}

func (caseConf *TestTradeConfig) RunTest(caseFile string, wg *sync.WaitGroup) {

	defer wg.Done()

	fLog := fileLog.New("module", "Trade")
	tLog := stdLog.New("module", "Trade")
	if _, err := toml.DecodeFile(caseFile, &caseConf); err != nil {

		tLog.Error("ErrTomlDecode", "Error", err.Error())
		return
	}

	tester := testcase.NewTestOperator(fLog, tLog)

	go tester.AddCaseArray(caseConf.TokenPreCreateCaseArr, caseConf.TokenFinishCreateCaseArr,
		caseConf.TransferCaseArr, caseConf.SellCaseArr, caseConf.DependBuyCaseArr)
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()
	tester.WaitTest()

}
