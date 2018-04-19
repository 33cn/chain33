package commands

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/types"
)

func CreateAccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new account with label",
		Run:   createAccount,
	}
	addCreateAccountFlags(cmd)
	return cmd
}

func addCreateAccountFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("label", "l", "", "account label")
	cmd.MarkFlagRequired("label")
}

func createAccount(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	label, _ := cmd.Flags().GetString("label")
	params := types.ReqNewAccount{
		Label: label,
	}
	var res types.WalletAccount
	ctx := NewRPCCtx(rpcLaddr, "Chain33.NewAccount", params, &res)
	ctx.SetResultCb(parseCreateAccountRes)
	ctx.Run()
}

func parseCreateAccountRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.WalletAccount)
	accResult := decodeAccount(res.GetAcc(), types.Coin)
	result := WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}
