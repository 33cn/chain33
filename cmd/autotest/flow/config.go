package flow

import (
	"gitlab.33.cn/chain33/chain33/cmd/autotest/types"
	"time"

	"reflect"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/inconshreveable/log15"
	. "gitlab.33.cn/chain33/chain33/cmd/autotest/types"
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
	CheckSleepTime  int            `toml:"checkSleepTime"`
	CheckTimeout    int            `toml:"checkTimeout"`
	TestCaseFileArr []TestCaseFile `toml:"TestCaseFile"`
}


type autoTestResult struct {

	dapp string
	totalCase int
	failCase int
	failCaseID []string
}


var (
	configFile string
	resultChan = make(chan *autoTestResult, 1)
	testResultArr = make([]*autoTestResult, 1)
	autoTestConfig = &TestCaseConfig{}

)


type TestRunner interface {
	RunTest(tomlFile string, wg *sync.WaitGroup)
}

func InitFlowConfig(conf string, log string) {

	fileLog.SetHandler(log15.Must.FileHandler(log, AutoTestLogFormat()))
	configFile = conf

}

//func DoTestOperation(configFile string) {
//
//	var wg sync.WaitGroup
//	var configConf TestCaseConfig
//
//	if _, err := toml.DecodeFile(configFile, &configConf); err != nil {
//
//		stdLog.Error("DecodeConfigFile", "filename", configFile, "Error", err.Error())
//		return
//	}
//
//	testcase.Init(configConf.CliCommand, configConf.CheckSleepTime, configConf.CheckTimeout)
//
//	stdLog.Info("[================================BeginAutoTest===============================]")
//	fileLog.Info("[================================BeginAutoTest===============================]")
//
//	for _, caseFile := range configConf.TestCaseFileArr {
//
//		filename := caseFile.Filename
//
//		switch caseFile.Dapp {
//
//		case "init":
//
//			//init需要优先进行处理, 阻塞等待完成
//			new(TestInitConfig).RunTest(filename, &wg)
//
//		case "bty":
//
//			wg.Add(1)
//			go new(TestBtyConfig).RunTest(filename, &wg)
//
//		case "token":
//
//			wg.Add(1)
//			go new(TestTokenConfig).RunTest(filename, &wg)
//
//		case "trade":
//
//			wg.Add(1)
//			go new(TestTradeConfig).RunTest(filename, &wg)
//
//		case "privacy":
//
//			wg.Add(1)
//			go new(TestPrivacyConfig).RunTest(filename, &wg)
//
//		}
//
//	}
//
//	wg.Wait()
//
//	stdLog.Info("[================================EndAutoTest=================================]")
//	fileLog.Info("[================================EndAutoTest=================================]")
//}
//
//





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
	types.Init(autoTestConfig.CliCommand, autoTestConfig.CheckTimeout)


	for _, caseFile := range autoTestConfig.TestCaseFileArr {

		wg.Add(1)
		go newTestFlow(caseFile.Dapp, caseFile.Filename, &wg)
	}

	//collect test results
	go func() {
		for {
			res, more := <- resultChan
			if more {
				testResultArr = append(testResultArr, res)
			}else {
				return
			}
		}

	}()

	wg.Wait()
	close(resultChan)
	time.Sleep(1 * time.Second)

	//log with test result
	bSuccess := true
	stdLog.Info("====================================AutoTestResult======================================================")
	fileLog.Info("====================================AutoTestResult======================================================")
	for _, res := range testResultArr {

		if res.failCase > 0 {

			bSuccess = false
			stdLog.Error("TestFailed", "dapp", res.dapp, "TotalCase", res.totalCase, "TotalFail", res.failCase, "FailID", res.failCaseID)
			fileLog.Error("TestFailed", "dapp", res.dapp, "TotalCase", res.totalCase, "TotalFail", res.failCase, "FailID", res.failCaseID)
		}else {

			stdLog.Error("TestSuccess", "dapp", res.dapp, "TotalCase", res.totalCase)
			fileLog.Error("TestSuccess", "dapp", res.dapp, "TotalCase", res.totalCase)
		}
	}

	return bSuccess
}




func newTestFlow(dapp string, filename string, wg *sync.WaitGroup) {



	defer wg.Done()


	configType := types.GetAutoTestConfig(dapp)
	caseConf := reflect.New(configType)

	if _, err := toml.DecodeFile(filename, caseConf.Addr().Interface()); err != nil {

		stdLog.Error("TomlDecodeTestCaseFile", "dapp", dapp, "Filename", filename, "Error", err.Error())
		return
	}


	//get config fields
	caseArrList := make([]interface{}, caseConf.NumField())

	for i := 0; i < caseConf.NumField(); i++ {

		caseArrList[i] = caseConf.Field(i).Interface()
	}

	tester := NewTestOperator(stdLog, fileLog, dapp)

	go tester.AddCaseArray(caseArrList...)
	go tester.HandleDependency()
	go tester.RunSendFlow()
	go tester.RunCheckFlow()


	testRes := tester.WaitTest()
	resultChan <- testRes

}

