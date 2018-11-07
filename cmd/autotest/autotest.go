package main

import (
	"flag"
	"fmt"
	"os"

	"gitlab.33.cn/chain33/chain33/cmd/autotest/flow"

	//新增dapp的autoest需要在此处导入对应autotest包，匿名注册
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/autotest"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/token/autotest"
	_ "gitlab.33.cn/chain33/chain33/plugin/dapp/trade/autotest"
	_ "gitlab.33.cn/chain33/chain33/system/dapp/coins/autotest"
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
