package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func SetFeeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set_fee",
		Short: "Set transaction fee",
		Run:   setFee,
	}
	addSetFeeFlags(cmd)
	return cmd
}

func addSetFeeFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("amount", "a", 0, "tx fee amount")
	cmd.MarkFlagRequired("amount")
}

func setFee(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount, _ := cmd.Flags().GetInt64("amount")
	params := types.ReqWalletSetFee{
		Amount: amount,
	}
	var res jsonrpc.Reply
	ctx := NewRPCCtx(rpcLaddr, "Chain33.SetTxFee", params, &res)
	ctx.Run()
}
