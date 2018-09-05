package contract

import (
	"sync"

	"github.com/BurntSushi/toml"
	"gitlab.33.cn/chain33/chain33/cmd/autotest/testcase"
)

type TestInitConfig struct {
	SimpleCaseArr            []testcase.SimpleCase            `toml:"SimpleCase,omitempty"`
	TransferCaseArr          []testcase.TransferCase          `toml:"TransferCase,omitempty"`
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

	/*for i := range caseConf.SimpleCaseArr {

		tester.AddCase(&caseConf.SimpleCaseArr[i])
	}

	for i := range caseConf.TransferCaseArr {

		tester.AddCase(&caseConf.TransferCaseArr[i])
	}

	for i := range caseConf.TokenPreCreateCaseArr {

		tester.AddCase(&caseConf.TokenPreCreateCaseArr[i])
	}

	for i := range caseConf.TokenFinishCreateCaseArr {

		tester.AddCase(&caseConf.TokenFinishCreateCaseArr[i])
	}*/

	tester.WaitTest()
}
