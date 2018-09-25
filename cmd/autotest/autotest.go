package main

import (
	"flag"

	"gitlab.33.cn/chain33/chain33/cmd/autotest/contract"
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

	contract.InitConfig(logFile)
	contract.DoTestOperation(configFile)

}
