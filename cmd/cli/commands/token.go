package commands

import (
	"github.com/spf13/cobra"
)

func TokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "token",
		Short: "Token managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		TokenBalanceCmd(),
		TokenPreCreateCmd(),
		TokenFinishCreateCmd(),
		TokenRevokeCreateCmd(),
		ListTokenCmd(),
		SellTokenCmd(),
		RevokeSellTokenCmd(),
		BuyTokenCmd(),
		ShowTokenOrderCmd(),

		TokenAssetsCmd(),
	)

	return cmd
}
