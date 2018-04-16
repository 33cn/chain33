package commands

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func BalanceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "balance",
		Short: "Get balance of a account address",
		Run:   balance,
	}
	addBalanceFlags(cmd)
	return cmd
}

func addBalanceFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account addr")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("exec", "e", "", `executer name ("none", "coins", "hashlock", "retrieve", "ticket", "token" and "trade" supported)`)
}

func balance(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	execer, _ := cmd.Flags().GetString("exec")

	isExecer := false
	for _, e := range [7]string{"none", "coins", "hashlock", "retrieve", "ticket", "token", "trade"} {
		if e == execer {
			isExecer = true
			break
		}
	}
	if !isExecer {
		fmt.Println("only none, coins, hashlock, retrieve, ticket, token, trade supported")
		return
	}

	var addrs []string
	addrs = append(addrs, addr)
	params := types.ReqBalance{
		Addresses: addrs,
		Execer:    execer,
	}

	var res []*jsonrpc.Account
	ctx := NewRPCCtx(rpcLaddr, "Chain33.GetBalance", params, &res)
	ctx.SetResultCb(parseGetBalanceRes)
	ctx.Run()
}

func parseGetBalanceRes(arg interface{}) (interface{}, error) {
	res := *arg.(*[]*jsonrpc.Account)
	balanceResult := strconv.FormatFloat(float64(res[0].Balance)/float64(types.Coin), 'f', 4, 64)
	frozenResult := strconv.FormatFloat(float64(res[0].Frozen)/float64(types.Coin), 'f', 4, 64)
	result := &AccountResult{
		Addr:     res[0].Addr,
		Currency: res[0].Currency,
		Balance:  balanceResult,
		Frozen:   frozenResult,
	}
	return result, nil
}
