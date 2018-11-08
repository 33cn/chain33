package main

import (
	"flag"
	"fmt"
	"os"

	"gitlab.33.cn/chain33/chain33/cmd/autotest/flow"
	//导入所有dapp的AutoTest
	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

var (
	configFile string
	logFile    string
)

func init() {

	flag.StringVar(&configFile, "f", "autotest.toml", "-f configFile")
	flag.StringVar(&logFile, "l", "autotest.log", "-l logFile")
	flag.Parse()
}

func main() {

	flow.InitFlowConfig(configFile, logFile)

	if flow.StartAutoTest() {

		fmt.Println("========================================Succeed!============================================")
		os.Exit(0)
	} else {
		fmt.Println("==========================================Failed!============================================")
		os.Exit(1)
	}
}
