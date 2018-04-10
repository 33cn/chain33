package commands

import (
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func RevokeSellTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke_sell",
		Short: "Revoke sold token",
		Run:   revokeSellToken,
	}
	addRevokeSellTokenFlags(cmd)
	return cmd
}

func addRevokeSellTokenFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("seller", "s", "", "token owner address")
	cmd.MarkFlagRequired("seller")

	cmd.Flags().StringP("sellid", "i", "", "sellid sold by seller")
	cmd.MarkFlagRequired("sellid")
}

func revokeSellToken(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	seller, _ := cmd.Flags().GetString("seller")
	sellId, _ := cmd.Flags().GetString("sellid")
	params := &types.ReqRevokeSell{
		Revoke: &types.TradeForRevokeSell{
			Sellid: sellId,
		},
		Owner: seller,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.RevokeSellToken", params, &res)
	ctx.Run()
}
