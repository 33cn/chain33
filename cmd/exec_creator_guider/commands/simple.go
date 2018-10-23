package commands

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cmd/exec_creator_guider/patterns"
)

func SimpleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simple",
		Short: "Create simple executor project mode",
		Run:   simpleCreate,
	}
	addSimpleCreateFlag(cmd)
	return cmd
}

func addSimpleCreateFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "executor project and class name")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringP("templatefile", "t", "", "template file path")

}

func simpleCreate(cmd *cobra.Command, args []string) {
	projectName, _ := cmd.Flags().GetString("name")
	className := projectName
	templateFile, _ := cmd.Flags().GetString("templatefile")
	if len(templateFile) == 0 {
		templateFile = fmt.Sprintf("template/%s.proto", projectName)
	}
	cp := patterns.New("simple")
	if cp == nil {
		return
	}
	cp.Run(projectName, className, "", "", templateFile)
}
