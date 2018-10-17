package commands

import (
	"fmt"
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/cmd/exec_creator_guider/patterns"
)

func AdvanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "advance",
		Short: "Create executor project in advance mode",
		Run:   advanceCreate,
	}
	addAdvanceCreateFlag(cmd)
	return cmd
}

func addAdvanceCreateFlag(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "executor project and class name")
	cmd.MarkFlagRequired("name")
	cmd.Flags().StringP("action", "a", "", "executor action class name")
	cmd.Flags().StringP("propfile", "p", "", "protobuf file path")
	cmd.Flags().StringP("templatefile", "t", "", "template file path")
}

func advanceCreate(cmd *cobra.Command, args []string) {
	projectName, _ := cmd.Flags().GetString("name")
	className := projectName
	actionName, _ := cmd.Flags().GetString("action")
	if len(actionName) == 0 {
		actionName = className + "Action"
	}
	propFile, _ := cmd.Flags().GetString("propfile")
	if len(propFile) == 0 {
		propFile = fmt.Sprintf("template/%s.proto", projectName)
	}
	templateFile, _ := cmd.Flags().GetString("templatefile")
	if len(templateFile) == 0 {
		templateFile = fmt.Sprintf("template/%s.template", projectName)
	}
	cp := patterns.New("advance")
	if cp == nil {
		return
	}
	cp.Run(projectName, className, actionName, propFile, templateFile)
}
