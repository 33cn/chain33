package commands

import "github.com/spf13/cobra"


func ImportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:"import",
		Short:"import plugin package",
		Run:importPackage,
	}
	return cmd
}

func importPackage(cmd *cobra.Command, args []string) {

}
