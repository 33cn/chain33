package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
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
	rpc, err := jsonrpc.NewJSONClient(rpcLaddr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	var res types.ReplyTokens
	err = rpc.Call("Chain33.Query", params, &res)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	for i, createdToken := range res.Tokens {
		createdToken.Price = createdToken.Price / types.Coin
		createdToken.Total = createdToken.Total / types.TokenPrecision

		fmt.Printf("---The %dth token is below--------------------\n", i)
		data, err := json.MarshalIndent(createdToken, "", "    ")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		fmt.Println(string(data))
	}
}
