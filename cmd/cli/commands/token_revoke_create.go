package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func TokenRevokeCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "revoke_create",
		Short: "Revoke creating a token",
		Run:   tokenRevokeCreate,
	}
	addTokenRevokeCreateFlags(cmd)
	return cmd
}

func addTokenRevokeCreateFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("creator", "c", "", "token creator address")
	cmd.MarkFlagRequired("creator")

	cmd.Flags().StringP("owner", "o", "", "token owner address")
	cmd.MarkFlagRequired("owner")

	cmd.Flags().StringP("symbol", "s", "", "token symbol")
	cmd.MarkFlagRequired("symbol")
}

func tokenRevokeCreate(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	creator, _ := cmd.Flags().GetString("creator")
	owner, _ := cmd.Flags().GetString("owner")
	symbol, _ := cmd.Flags().GetString("symbol")
	params := types.ReqTokenRevokeCreate{
		RevokerAddr: creator,
		Symbol:      symbol,
		OwnerAddr:   owner,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRPCCtx(rpcLaddr, "Chain33.TokenRevokeCreate", params, &res)
	ctx.Run()
}
