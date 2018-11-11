// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"

	"github.com/33cn/chain33/cmd/autotest/contract"
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
