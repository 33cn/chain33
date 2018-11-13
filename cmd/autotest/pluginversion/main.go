// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//本地存在github.com/33cn/plugin官方包时，autotest编译版本
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/33cn/chain33/cmd/autotest/testflow"
	//导入所有dapp的AutoTest
	_ "github.com/33cn/chain33/system"
	_ "github.com/33cn/plugin/plugin/dapp/init"
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

	testflow.InitFlowConfig(configFile, logFile)

	if testflow.StartAutoTest() {

		fmt.Println("========================================Succeed!============================================")
		os.Exit(0)
	} else {
		fmt.Println("==========================================Failed!============================================")
		os.Exit(1)
	}
}
