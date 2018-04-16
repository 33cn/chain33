package commands

import (
	"github.com/spf13/cobra"
)

func KeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Key managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ImportKeyCmd(),
		DumpKeyCmd(),
	)

	return cmd
}
