package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cmd/tools/strategy"
	"gitlab.33.cn/chain33/chain33/cmd/tools/types"
)

func ImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import plugin package",
		Run:   importPackage,
	}
	cmd.Flags().StringP("conf", "c", "plugin/plugin.toml", "path of plugin config file")
	cmd.Flags().StringP("path", "p", "plugin", "path of plugin")
	cmd.Flags().StringP("packname", "", "", "project package name")
	return cmd
}

func importPackage(cmd *cobra.Command, args []string) {
	s := strategy.New(types.KeyImportPackage)
	if s == nil {
		fmt.Println(types.KeyImportPackage, "Not support")
		return
	}
	conf, _ := cmd.Flags().GetString("conf")
	path, _ := cmd.Flags().GetString("path")
	packname, _ := cmd.Flags().GetString("packname")
	s.SetParam("path", path)
	s.SetParam("packname", packname)
	s.SetParam("conf", conf)
	s.Run()
}
