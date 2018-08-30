package contract

import (
	"sync"

	"github.com/BurntSushi/toml"
	"gitlab.33.cn/chain33/chain33/cmd/autotest/testcase"
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
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()

	for i := range caseConf.TokenPreCreateCaseArr {

		tester.AddCase(&caseConf.TokenPreCreateCaseArr[i])
	}

	for i := range caseConf.TokenFinishCreateCaseArr {

		tester.AddCase(&caseConf.TokenFinishCreateCaseArr[i])
	}

	for i := range caseConf.TransferCaseArr {

		tester.AddCase(&caseConf.TransferCaseArr[i])
	}

	for i := range caseConf.SellCaseArr {

		tester.AddCase(&caseConf.SellCaseArr[i])
	}

	for i := range caseConf.DependBuyCaseArr {

		tester.AddCase(&caseConf.DependBuyCaseArr[i])
	}

	tester.WaitTest()

}
