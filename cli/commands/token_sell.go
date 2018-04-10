package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func SellTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sell",
		Short: "Sell token",
		Run:   sellToken,
	}
	addSellTokenFlags(cmd)
	return cmd
}

func addSellTokenFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("owner", "o", "", "token owner address")
	cmd.MarkFlagRequired("owner")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().Float64P("amount", "a", 0, "amount board lot")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().Float64P("price", "p", 0, "price board lot")
	cmd.MarkFlagRequired("price")

	cmd.Flags().Int64P("min", "m", 0, "min board lot")
	cmd.MarkFlagRequired("min")

	cmd.Flags().Int64P("total", "t", 0, "total board lot")
	cmd.MarkFlagRequired("total")

	cmd.Flags().Int64("start", 0, "start time")
	cmd.Flags().Int64("end", 0, "end time")
	cmd.Flags().Bool("crowd", false, "whether crowd found")
}

func sellToken(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	owner, _ := cmd.Flags().GetString("owner")
	symbol, _ := cmd.Flags().GetString("symbol")
	amount, _ := cmd.Flags().GetFloat64("amount")
	price, _ := cmd.Flags().GetFloat64("price")
	min, _ := cmd.Flags().GetInt64("min")
	total, _ := cmd.Flags().GetInt64("total")
	start, _ := cmd.Flags().GetInt64("start")
	end, _ := cmd.Flags().GetInt64("end")
	isCrowFund, _ := cmd.Flags().GetBool("crowd")
	params := &types.ReqSellToken{
		Sell: &types.TradeForSell{
			Tokensymbol:       symbol,
			Amountperboardlot: int64(amount*types.InputPrecision) * types.Multiple1E4,
			Minboardlot:       min,
			Priceperboardlot:  int64(price*types.InputPrecision) * types.Multiple1E4,
			Totalboardlot:     total,
			Starttime:         start,
			Stoptime:          end,
			Crowdfund:         isCrowFund,
		},
		Owner: owner,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SellToken", params, &res)
	ctx.Run()
}
