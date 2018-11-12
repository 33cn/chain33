// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contract

import (
	"sync"

	"github.com/33cn/chain33/cmd/autotest/testcase"
	"github.com/BurntSushi/toml"
)

type TestPrivacyConfig struct {
	SimpleCaseArr            []testcase.SimpleCase            `toml:"SimpleCase,omitempty"`
	TokenPreCreateCaseArr    []testcase.TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []testcase.TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []testcase.TransferCase          `toml:"TransferCase,omitempty"`
	PubToPrivCaseArr         []testcase.PubToPrivCase         `toml:"PubToPrivCase,omitempty"`
	PrivToPrivCaseArr        []testcase.PrivToPrivCase        `toml:"PrivToPrivCase,omitempty"`
	PrivToPubCaseArr         []testcase.PrivToPubCase         `toml:"PrivToPubCase,omitempty"`
	PrivCreateutxosCaseArr   []testcase.PrivCreateutxosCase   `toml:"PrivCreateutxosCase,omitempty"`
}

func (caseConf *TestPrivacyConfig) RunTest(caseFile string, wg *sync.WaitGroup) {

	defer wg.Done()

	fLog := fileLog.New("module", "Privacy")
	tLog := stdLog.New("module", "Privacy")
	if _, err := toml.DecodeFile(caseFile, &caseConf); err != nil {

		tLog.Error("ErrTomlDecode", "Error", err.Error())
		return
	}
	tester := testcase.NewTestOperator(fLog, tLog)

	go tester.AddCaseArray(caseConf.SimpleCaseArr, caseConf.TokenPreCreateCaseArr, caseConf.TokenFinishCreateCaseArr,
		caseConf.TransferCaseArr, caseConf.PubToPrivCaseArr, caseConf.PrivToPrivCaseArr, caseConf.PrivToPubCaseArr, caseConf.PrivCreateutxosCaseArr)
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()
	tester.WaitTest()
}
