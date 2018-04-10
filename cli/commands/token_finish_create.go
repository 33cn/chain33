package commands

import (
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func TokenFinishCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "finish_create",
		Short: "Finish creating a token",
		Run:   tokenFinishCreate,
	}
	addTokenFinishCreateFlags(cmd)
	return cmd
}

func addTokenFinishCreateFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("creator", "c", "", "token creator address")
	cmd.MarkFlagRequired("creator")

	cmd.Flags().StringP("owner", "o", "", "token owner address")
	cmd.MarkFlagRequired("owner")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")
}

func tokenFinishCreate(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	creator, _ := cmd.Flags().GetString("creator")
	owner, _ := cmd.Flags().GetString("owner")
	symbol, _ := cmd.Flags().GetString("symbol")
	params := types.ReqTokenFinishCreate{
		FinisherAddr: creator,
		Symbol:       symbol,
		OwnerAddr:    owner,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.TokenFinishCreate", params, &res)
	ctx.Run()
}
