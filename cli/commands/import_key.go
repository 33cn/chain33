package commands

import (
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

func ImportKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "Import private key with label",
		Run:   importKey,
	}
	addImportKeyFlags(cmd)
	return cmd
}

func addImportKeyFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "private key")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("label", "l", "", "label for private key")
	cmd.MarkFlagRequired("label")
}

func importKey(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	key, _ := cmd.Flags().GetString("key")
	label, _ := cmd.Flags().GetString("label")
	params := types.ReqWalletImportPrivKey{
		Privkey: key,
		Label:   label,
	}
	var res types.WalletAccount
	ctx := NewRpcCtx(rpcLaddr, "Chain33.ImportPrivkey", params, &res)
	ctx.SetResultCb(parseImportKeyRes)
	ctx.Run()
}

func parseImportKeyRes(arg interface{}) (interface{}, error) {
	res := arg.(*types.WalletAccount)
	accResult := decodeAccount(res.GetAcc(), types.Coin)
	result := WalletResult{
		Acc:   accResult,
		Label: res.GetLabel(),
	}
	return result, nil
}
