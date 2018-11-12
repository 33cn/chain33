// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package contract

import (
	"sync"

	"github.com/33cn/chain33/cmd/autotest/testcase"
	"github.com/33cn/chain33/common/log/log15"
	"github.com/BurntSushi/toml"
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
	Contract string `toml:"contract"`
	Filename string `toml:"filename"`
}

type TestCaseConfig struct {
	CliCommand      string         `toml:"cliCmd"`
	CheckSleepTime  int            `toml:"checkSleepTime"`
	CheckTimeout    int            `toml:"checkTimeout"`
	TestCaseFileArr []TestCaseFile `toml:"TestCaseFile"`
}

type TestRunner interface {
	RunTest(tomlFile string, wg *sync.WaitGroup)
}

func InitConfig(logfile string) {

	fileLog.SetHandler(log15.Must.FileHandler(logfile, testcase.AutoTestLogFormat()))
	stdLog.SetHandler(log15.StdoutHandler)

}

func DoTestOperation(configFile string) {

	var wg sync.WaitGroup
	var configConf TestCaseConfig

	if _, err := toml.DecodeFile(configFile, &configConf); err != nil {

		stdLog.Error("DecodeConfigFile", "filename", configFile, "Error", err.Error())
		return
	}

	testcase.Init(configConf.CliCommand, configConf.CheckSleepTime, configConf.CheckTimeout)

	stdLog.Info("[================================BeginAutoTest===============================]")
	fileLog.Info("[================================BeginAutoTest===============================]")

	for _, caseFile := range configConf.TestCaseFileArr {

		filename := caseFile.Filename

		switch caseFile.Contract {

		case "init":

			//init需要优先进行处理, 阻塞等待完成
			new(TestInitConfig).RunTest(filename, &wg)

		case "bty":

			wg.Add(1)
			go new(TestBtyConfig).RunTest(filename, &wg)

		case "token":

			wg.Add(1)
			go new(TestTokenConfig).RunTest(filename, &wg)

		case "trade":

			wg.Add(1)
			go new(TestTradeConfig).RunTest(filename, &wg)

		case "privacy":

			wg.Add(1)
			go new(TestPrivacyConfig).RunTest(filename, &wg)

		}

	}

	wg.Wait()

	stdLog.Info("[================================EndAutoTest=================================]")
	fileLog.Info("[================================EndAutoTest=================================]")
}
