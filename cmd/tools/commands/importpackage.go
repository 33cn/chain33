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

//ImportCmd 导入命令
func ImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import plugin package",
		Run:   importPackage,
	}
	cmd.Flags().StringP("conf", "c", "plugin/plugin.toml", "path of plugin config file")
	cmd.Flags().StringP("path", "p", "plugin", "path of plugin")
	cmd.Flags().StringP("out", "o", "", "output new config file")
	cmd.Flags().StringP("packname", "", "", "project package name")
	return cmd
}

//import 之前默认做处理
func importPackage(cmd *cobra.Command, args []string) {
	out, _ := cmd.Flags().GetString("out")
	conf, _ := cmd.Flags().GetString("conf")
	path, _ := cmd.Flags().GetString("path")
	packname, _ := cmd.Flags().GetString("packname")
	s := strategy.New(types.KeyUpdateInit)
	if s == nil {
		fmt.Println(types.KeyUpdateInit, "Not support")
		return
	}
	s.SetParam("path", path)
	s.SetParam("packname", packname)
	s.Run()

	s = strategy.New(types.KeyImportPackage)
	if s == nil {
		fmt.Println(types.KeyImportPackage, "Not support")
		return
	}
	s.SetParam("path", path)
	s.SetParam("packname", packname)
	s.SetParam("conf", conf)
	s.SetParam("out", out)
	s.Run()
}
