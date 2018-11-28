// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package 自动化系统回归测试工具，外部支持输入测试用例配置文件，
// 输出测试用例执行结果并记录详细执行日志。
// 内部代码支持用例扩展开发，继承并实现通用接口，即可自定义实现用例类型。
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/33cn/chain33/cmd/autotest/testflow"
	//默认只导入系统dapp的AutoTest
	_ "github.com/33cn/chain33/system"
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
