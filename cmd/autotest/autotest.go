package main

import (
	"flag"
	"fmt"
	"os"

	"gitlab.33.cn/chain33/chain33/cmd/autotest/flow"

	_ "gitlab.33.cn/chain33/chain33/cmd/autotest/autotest"
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

		fmt.Println("Success!")
		os.Exit(0)
	}else {
		fmt.Println("Failed!")
		os.Exit(1)
	}
}
