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

}

func createPlugin(cmd *cobra.Command, args []string) {

	s := strategy.New(types.KeyCreatePlugin)
	if s == nil {
		fmt.Println(types.KeyCreatePlugin, "Not support")
		return
	}
	s.Run()
}
