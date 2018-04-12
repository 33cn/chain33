package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func BuyTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "buy",
		Short: "Buy token",
		Run:   buyToken,
	}
	addBuyTokenFlags(cmd)
	return cmd
}

func addBuyTokenFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("buyer", "b", "", "buyer address")
	cmd.MarkFlagRequired("buyer")

	cmd.Flags().StringP("sellid", "s", "", "sell id")
	cmd.MarkFlagRequired("sellid")

	cmd.Flags().Int64P("amount", "c", 0, "amount to buy")
	cmd.MarkFlagRequired("amount")
}

func buyToken(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	buyer, _ := cmd.Flags().GetString("buyer")
	sellID, _ := cmd.Flags().GetString("sellid")
	amount, _ := cmd.Flags().GetInt64("amount")
	params := &types.ReqBuyToken{
		Buy: &types.TradeForBuy{
			Sellid:      sellID,
			Boardlotcnt: amount,
		},
		Buyer: buyer,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRPCCtx(rpcLaddr, "Chain33.BuyToken", params, &res)
	ctx.Run()
}
