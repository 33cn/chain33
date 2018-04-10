package commands

import (
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"github.com/spf13/cobra"
)

func SendTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a transaction",
		Run:   sendTx,
	}
	addSendTxFlags(cmd)
	return cmd
}

func addSendTxFlags(cmd *cobra.Command) {
	// TODO, base64 format?
	cmd.Flags().StringP("data", "d", "", "transaction content")
	cmd.MarkFlagRequired("data")
}

func sendTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	data, _ := cmd.Flags().GetString("data")
	params := jsonrpc.RawParm{
		Data: data,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.SendTransaction", params, &res)
	ctx.Run()
}
