package commands

import (
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func TokenPreCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pre_create",
		Short: "Precreate a token",
		Run:   tokenPreCreate,
	}
	addTokenPreCreateFlags(cmd)
	return cmd
}

func addTokenPreCreateFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("creator", "c", "", "token creator address")
	cmd.MarkFlagRequired("creator")

	cmd.Flags().StringP("owner", "o", "", "token owner address")
	cmd.MarkFlagRequired("owner")

	cmd.Flags().StringP("name", "n", "", "token name")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")

	cmd.Flags().StringP("intro", "i", "", "token introduction")
	cmd.MarkFlagRequired("intro")

	cmd.Flags().Int64P("total", "t", 0, "total supply of token")
	cmd.MarkFlagRequired("total")

	cmd.Flags().Int64P("price", "p", 0, "token price")
	cmd.MarkFlagRequired("price")
}

func tokenPreCreate(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	creator, _ := cmd.Flags().GetString("creator")
	owner, _ := cmd.Flags().GetString("owner")
	name, _ := cmd.Flags().GetString("name")
	symbol, _ := cmd.Flags().GetString("symbol")
	intro, _ := cmd.Flags().GetString("intro")
	total, _ := cmd.Flags().GetInt64("total")
	price, _ := cmd.Flags().GetInt64("price")
	params := types.ReqTokenPreCreate{
		CreatorAddr:  creator,
		Name:         name,
		Symbol:       symbol,
		Introduction: intro,
		OwnerAddr:    owner,
		Total:        total * types.TokenPrecision,
		Price:        price * types.Coin,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.TokenPreCreate", params, &res)
	ctx.Run()
}
