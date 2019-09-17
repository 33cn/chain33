// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
)

func init() {

	base.RegisterCodeFile(cmdCodeFile{})
}

type cmdCodeFile struct {
	base.DappCodeFile
}

func (cmdCodeFile) GetDirName() string {

	return "cmd"
}

func (cmdCodeFile) GetFiles() map[string]string {

	return map[string]string{
		buildShellName: buildShellContent,
		makeFIleName:   makeFileContent,
	}
}

var (
	buildShellName    = "build.sh"
	buildShellContent = `#!/bin/sh
# 官方ci集成脚本
strpwd=$(pwd)
strcmd=${strpwd##*dapp/}
strapp=${strcmd%/cmd*}

OUT_DIR="${1}/$strapp"
#FLAG=$2

mkdir -p "${OUT_DIR}"
cp ./build/* "${OUT_DIR}"
`
	makeFIleName    = "Makefile"
	makeFileContent = `all:
	chmod +x ./build.sh
	./build.sh $(OUT) $(FLAG)
`
)
