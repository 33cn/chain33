package commands

import (
	"github.com/spf13/cobra"
)

func TxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "Transaction managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		SetFeeCmd(),
		CreateTxCmd(),
		SendTxCmd(),
		QueryTxByHashCmd(),
		QueryTxByAddrCmd(),
		DecodeTxCmd(),
	)

	return cmd
}
