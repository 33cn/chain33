package commands

import (
	"fmt"

	"gitlab.33.cn/chain33/chain33/cmd/tools/strategy"
	"gitlab.33.cn/chain33/chain33/cmd/tools/types"

	"github.com/spf13/cobra"
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

	s := strategy.New(types.KeyCreateSimpleExecProject)
	if s == nil {
		fmt.Println(types.KeyCreateSimpleExecProject, "Not support")
		return
	}
	s.SetParam(types.KeyProjectName, projectName)
	s.SetParam(types.KeyClassName, className)
	s.SetParam(types.KeyTemplateFilePath, templateFile)
	s.Run()
}
