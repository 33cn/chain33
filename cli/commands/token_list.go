package commands

import (
	"fmt"
	"os"

	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func ListTokenCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List precreated or finishcreated token",
		Run:   listToken,
	}
	addListTokenFlags(cmd)
	return cmd
}

func addListTokenFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("status", "s", "", "token status ('pre' or 'created')")
	cmd.MarkFlagRequired("status")
}

func listToken(cmd *cobra.Command, args []string) {
	status, _ := cmd.Flags().GetString("status")
	var reqtokens types.ReqTokens
	reqtokens.Queryall = true
	if status == "pre" {
		reqtokens.Status = types.TokenStatusPreCreated
	} else if status == "created" {
		reqtokens.Status = types.TokenStatusCreated
	} else {
		fmt.Fprintln(os.Stderr, "invalid status")
		return
	}

	var params jsonrpc.Query4Cli
	params.Execer = "token"
	params.FuncName = "GetTokens"
	params.Payload = reqtokens

	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res types.ReplyTokens
	ctx := NewRpcCtx(rpcLaddr, "Chain33.Query", params, &res)
	ctx.SetResultCb(parseTokenListRes)
	ctx.Run()
}

func parseTokenListRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.ReplyTokens)
	var result []*types.Token
	for _, preCreatedToken := range res.Tokens {
		one := &types.Token{
			Price: preCreatedToken.Price / types.Coin,
			Total: preCreatedToken.Total / types.TokenPrecision,
		}
		result = append(result, one)
	}
	return result, nil
}
