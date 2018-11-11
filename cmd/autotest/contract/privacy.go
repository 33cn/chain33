package contract

import (
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/33cn/chain33/cmd/autotest/testcase"
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
