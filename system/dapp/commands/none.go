package commands

import (
	"fmt"
	"os"

	cmdtypes "github.com/33cn/chain33/system/dapp/commands/types"
	nonetypes "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/spf13/cobra"
)

// NoneCmd dapp none command
func NoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "none",
		Short: "dapp none",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		commitDelayTxCmd(),
	)
	return cmd
}

// createDelayTx create commit delay transaction
func commitDelayTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delay",
		Short: "create commit delay tx",
		Run:   createCommintDelayTx,
	}
	addCommitDelayFlags(cmd)
	return cmd
}

func addCommitDelayFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("delaytx", "d", "", "delay transaction data(hex format)")
	cmd.MarkFlagRequired("delaytx")

	cmd.Flags().Int64P("delaytime", "t", 10, "relative delay time(seconds)")
	cmd.MarkFlagRequired("delaytime")
}

func createCommintDelayTx(cmd *cobra.Command, args []string) {
	delayTx, _ := cmd.Flags().GetString("delaytx")
	delayTime, _ := cmd.Flags().GetInt64("delaytime")

	if delayTx == "" {
		fmt.Fprintf(os.Stderr, "empty delay tx param")
		return
	}

	if delayTime < 1 {
		fmt.Fprintf(os.Stderr, "negetive delay height param")
		return
	}

	payload := &nonetypes.CommitDelayTx{
		DelayTx:           delayTx,
		RelativeDelayTime: delayTime,
	}
	cmdtypes.SendCreateTxRPC(cmd, nonetypes.NoneX, nonetypes.NameCommitDelayTxAction, payload)
}
