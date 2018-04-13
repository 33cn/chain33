package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func AccountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account managerment",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		ListAccountCmd(),
		CreateAccountCmd(),
		MergeBalanceCmd(),
		TransferCmd(),
		SetLabelCmd(),
		SetPwdCmd(),
		BalanceCmd(),
	)

	return cmd
}

// set_label
func SetLabelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set_label",
		Short: "Set label for account address",
		Run:   setLabel,
	}
	addSetLabelFlags(cmd)
	return cmd
}

func addSetLabelFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("label", "l", "", "account label")
	cmd.MarkFlagRequired("label")
}

func setLabel(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	label, _ := cmd.Flags().GetString("label")
	params := types.ReqWalletSetLabel{
		Addr:  addr,
		Label: label,
	}
	var res types.WalletAccount
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SetLabl", params, &res)
	ctx.SetResultCb(parseSetLabelRes)
	ctx.Run()
}

func parseSetLabelRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.WalletAccount)
	accResult := decodeAccount(res.GetAcc(), types.Coin)
	result := WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}

// set_pwd
func SetPwdCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set_pwd",
		Short: "Set password",
		Run:   setPwd,
	}
	addSetPwdFlags(cmd)
	return cmd
}

func addSetPwdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("old", "o", "", "old password")
	cmd.MarkFlagRequired("old")

	cmd.Flags().StringP("new", "n", "", "new password")
	cmd.MarkFlagRequired("new")
}

func setPwd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	oldPwd, _ := cmd.Flags().GetString("old")
	newPwd, _ := cmd.Flags().GetString("new")
	params := types.ReqWalletSetPasswd{
		Oldpass: oldPwd,
		Newpass: newPwd,
	}
	var res jsonrpc.Reply
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SetPasswd", params, &res)
	ctx.Run()
}
