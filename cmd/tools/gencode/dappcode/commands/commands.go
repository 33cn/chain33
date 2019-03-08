// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
)

func init() {

	base.RegisterCodeFile(commandsCodeFile{})
}


type commandsCodeFile struct {
	base.DappCodeFile
}


func (c commandsCodeFile)GetDirName() string {

	return "commands"
}


func (c commandsCodeFile)GetFiles() map[string]string {

	return map[string]string{
		commandsFileName: commandsFileContent,
	}
}




var (

	commandsFileName = "commands.go"
	commandsFileContent =
`/* package commands, implement dapp client commands*/
package commands

import "github.com/spf13/cobra"

func Cmd() *cobra.Command {
	return &cobra.Command{}
}
`
)