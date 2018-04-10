package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func MergeBalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merge",
		Short: "Merge accounts' balance into address",
		Run:   mergeBalance,
	}
	addMergeBalanceFlags(cmd)
	return cmd
}

func addMergeBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "destination account address")
	cmd.MarkFlagRequired("to")
}

func mergeBalance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	toAddr, _ := cmd.Flags().GetString("to")
	params := types.ReqWalletMergeBalance{
		To: toAddr,
	}
	var res jsonrpc.ReplyHashes
	ctx := NewRpcCtx(rpcLaddr, "Chain33.MergeBalance", params, &res)
	ctx.Run()
}
