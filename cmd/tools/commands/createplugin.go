//Copyright Fuzamei Corp. 2018 All Rights Reserved.
//Use of this source code is governed by a BSD-style
//license that can be found in the LICENSE file.

package commands

import (
	"fmt"

	"github.com/33cn/chain33/cmd/tools/strategy"
	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/spf13/cobra"
)

//CreatePluginCmd 构造插件命令
func CreatePluginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "createplugin",
		Short: "Create chain33 plugin project mode",
		Run:   createPlugin,
	}
	addCreatePluginFlag(cmd)
	return cmd
}

func addCreatePluginFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "project name")
	cmd.MarkFlagRequired("name")

}

func createPlugin(cmd *cobra.Command, args []string) {
	projectName, _ := cmd.Flags().GetString("name")

	s := strategy.New(types.KeyCreatePlugin)
	if s == nil {
		fmt.Println(types.KeyCreatePlugin, "Not support")
		return
	}
	s.SetParam(types.KeyProjectName, projectName)
	s.SetParam(types.KeyExecutorName, projectName)
	s.SetParam(types.KeyClassName, projectName)
	s.Run()
}
