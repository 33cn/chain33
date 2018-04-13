package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func TransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Transfer coin or token to address",
		Run:   transfer,
	}
	addTransferFlags(cmd)
	return cmd
}

func addTransferFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("from", "f", "", "sender account address")
	cmd.MarkFlagRequired("from")

	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.Flags().String("token", "", "token symbol")
}

func transfer(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	fromAddr, _ := cmd.Flags().GetString("from")
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	token, _ := cmd.Flags().GetString("token")
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	params := types.ReqWalletSendToAddress{
		From:        fromAddr,
		To:          toAddr,
		Amount:      amountInt64,
		Note:        note,
		TokenSymbol: token,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRPCCtx(rpcLaddr, "Chain33.SendToAddress", params, &res)
	ctx.Run()
}
