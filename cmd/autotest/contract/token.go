package contract

import (
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/33cn/chain33/cmd/autotest/testcase"
)

type TestTokenConfig struct {
	TokenPreCreateCaseArr    []testcase.TokenPreCreateCase    `toml:"TokenPreCreateCase,omitempty"`
	TokenFinishCreateCaseArr []testcase.TokenFinishCreateCase `toml:"TokenFinishCreateCase,omitempty"`
	TransferCaseArr          []testcase.TransferCase          `toml:"TransferCase,omitempty"`
	WithdrawCaseArr          []testcase.WithdrawCase          `toml:"WithdrawCase,omitempty"`
	TokenrevokeCaseArr       []testcase.TokenrevokeCase       `toml:"TokenrevokeCase,omitempty"`
}

func (caseConf *TestTokenConfig) RunTest(caseFile string, wg *sync.WaitGroup) {

	defer wg.Done()

	fLog := fileLog.New("module", "Token")
	tLog := stdLog.New("module", "Token")
	if _, err := toml.DecodeFile(caseFile, &caseConf); err != nil {

		tLog.Error("ErrTomlDecode", "Error", err.Error())
		return
	}

	tester := testcase.NewTestOperator(fLog, tLog)

	go tester.AddCaseArray(caseConf.TokenPreCreateCaseArr, caseConf.TokenFinishCreateCaseArr,
		caseConf.TransferCaseArr, caseConf.WithdrawCaseArr, caseConf.TokenrevokeCaseArr)
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()
	tester.WaitTest()

}
