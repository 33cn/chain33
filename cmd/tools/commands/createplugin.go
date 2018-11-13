package commands

import "github.com/spf13/cobra"

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

}
