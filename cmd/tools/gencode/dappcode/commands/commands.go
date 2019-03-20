// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(commandsCodeFile{})
}

type commandsCodeFile struct {
	base.DappCodeFile
}

func (c commandsCodeFile) GetDirName() string {

	return "commands"
}

func (c commandsCodeFile) GetFiles() map[string]string {

	return map[string]string{
		commandsFileName: commandsFileContent,
	}
}

func (c commandsCodeFile) GetFileReplaceTags() []string {
	return []string{types.TagExecName}
}

var (
	commandsFileName    = "commands.go"
	commandsFileContent = `/*Package commands implement dapp client commands*/
package commands

import (
	"github.com/spf13/cobra"
)

/* 
 * 实现合约对应客户端
*/

// Cmd ${EXECNAME} client command
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:	"${EXECNAME}",
		Short:	"${EXECNAME} command",
		Args:	cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		//add sub command
	)
	return cmd
}
`
)
