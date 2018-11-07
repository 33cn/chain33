package flow

import (
	"fmt"
	"time"

	"gitlab.33.cn/chain33/chain33/cmd/autotest/types"

	"reflect"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/inconshreveable/log15"
)

var (
	fileLog = log15.New()
	stdLog  = log15.New()
)

//contract type
/*
bty,
token,
trade,

*/

type TestCaseFile struct {
	Dapp     string `toml:"dapp"`
	Filename string `toml:"filename"`
}

type TestCaseConfig struct {
	CliCommand      string         `toml:"cliCmd"`
	CheckTimeout    int            `toml:"checkTimeout"`
	TestCaseFileArr []TestCaseFile `toml:"TestCaseFile"`
}

type autoTestResult struct {
	dapp       string
	totalCase  int
	failCase   int
	failCaseID []string
}

var (
	configFile     string
	resultChan     = make(chan *autoTestResult, 1)
	testResultArr  = make([]*autoTestResult, 0)
	autoTestConfig = &TestCaseConfig{}

	checkSleepTime = 1 //second, s
)

type TestRunner interface {
	RunTest(tomlFile string, wg *sync.WaitGroup)
}

func InitFlowConfig(conf string, log string) {

	fileLog.SetHandler(log15.Must.FileHandler(log, types.AutoTestLogFormat()))
	configFile = conf

}

func StartAutoTest() bool {

	stdLog.Info("[================================BeginAutoTest===============================]")
	fileLog.Info("[================================BeginAutoTest===============================]")
	var wg sync.WaitGroup

	if _, err := toml.DecodeFile(configFile, &autoTestConfig); err != nil {

		stdLog.Error("TomlDecodeAutoTestConfig", "Filename", configFile, "Err", err.Error())
		return false
	}

	if len(autoTestConfig.CliCommand) == 0 {

		stdLog.Error("NullChain33Cli")
		return false
	}
	//init types
	types.Init(autoTestConfig.CliCommand, autoTestConfig.CheckTimeout/checkSleepTime)

	for _, caseFile := range autoTestConfig.TestCaseFileArr {

		wg.Add(1)
		go newTestFlow(caseFile.Dapp, caseFile.Filename, &wg)
	}

	//collect test results
	go func() {
		for {
			res, more := <-resultChan
			if more {
				testResultArr = append(testResultArr, res)
			} else {
				return
			}
		}

	}()

	wg.Wait()
	close(resultChan)
	time.Sleep(1 * time.Second)

	//log with test result
	bSuccess := true
	fmt.Println("==================================AutoTestResultSummary======================================")
	fileLog.Info("====================================AutoTestResultSummary========================================")
	for _, res := range testResultArr {

		if res.failCase > 0 {

			bSuccess = false
			stdLog.Error("TestFailed", "dapp", res.dapp, "TotalCase", res.totalCase, "TotalFail", res.failCase, "FailID", res.failCaseID)
			fileLog.Error("TestFailed", "dapp", res.dapp, "TotalCase", res.totalCase, "TotalFail", res.failCase, "FailID", res.failCaseID)
		} else {

			stdLog.Info("TestSuccess", "dapp", res.dapp, "TotalCase", res.totalCase, "TotalFail", res.failCase)
			fileLog.Info("TestSuccess", "dapp", res.dapp, "TotalCase", res.totalCase, "TotalFail", res.failCase)
		}
	}

	return bSuccess
}

func newTestFlow(dapp string, filename string, wg *sync.WaitGroup) {

	defer wg.Done()

	configType := types.GetAutoTestConfig(dapp)

	if configType == nil {

		stdLog.Error("GetAutoTestConfigType", "DappName", dapp, "Error", "NotSupportAutoTestType")
		return
	}

	if configType.Kind() == reflect.Ptr {
		configType = configType.Elem()
	}

	caseConf := reflect.New(configType)

	if _, err := toml.DecodeFile(filename, caseConf.Interface()); err != nil {

		stdLog.Error("TomlDecodeTestCaseFile", "Dapp", dapp, "Filename", filename, "Error", err.Error())
		return
	}

	//get config fields
	fields := caseConf.Elem().NumField()
	caseArrList := make([]interface{}, fields)

	for i := 0; i < fields; i++ {

		caseArrList[i] = caseConf.Elem().Field(i).Interface()
	}

	tester := NewTestOperator(stdLog, fileLog, dapp)

	go tester.AddCaseArray(caseArrList...)
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()

	testRes := tester.WaitTest()
	resultChan <- testRes

}
