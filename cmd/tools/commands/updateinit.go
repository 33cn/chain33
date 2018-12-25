// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"

	"github.com/33cn/chain33/cmd/tools/strategy"
	"github.com/33cn/chain33/cmd/tools/types"
	"github.com/spf13/cobra"
)

//UpdateInitCmd 升级初始命令
func UpdateInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "updateinit",
		Short: "Update chain33 plugin consensus、dapp、store、mempool init.go file",
		Run:   updateInit,
	}
	cmd.Flags().StringP("path", "p", "plugin", "path of plugin")
	cmd.Flags().StringP("out", "o", "", "output new config file")
	cmd.Flags().StringP("packname", "", "", "project package name")
	return cmd
}

func updateInit(cmd *cobra.Command, args []string) {
	path, _ := cmd.Flags().GetString("path")
	packname, _ := cmd.Flags().GetString("packname")
	out, _ := cmd.Flags().GetString("out")

	s := strategy.New(types.KeyUpdateInit)
	if s == nil {
		fmt.Println(types.KeyUpdateInit, "Not support")
		return
	}
	s.SetParam("out", out)
	s.SetParam("path", path)
	s.SetParam("packname", packname)
	s.Run()
}
