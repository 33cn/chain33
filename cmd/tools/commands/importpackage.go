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
	return cmd
}

func importPackage(cmd *cobra.Command, args []string) {
	s := strategy.New(types.KeyImportPackage)
	if s == nil {
		fmt.Println(types.KeyImportPackage, "Not support")
		return
	}
	s.Run()
}