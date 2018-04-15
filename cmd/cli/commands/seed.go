package commands

import (
	"github.com/spf13/cobra"
)

func SeedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Seed managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GenSeedCmd(),
		SaveSeedCmd(),
		GetSeedCmd(),
	)

	return cmd
}
