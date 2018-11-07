package commands

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	pty "gitlab.33.cn/chain33/chain33/plugin/dapp/hashlock/types"
	"gitlab.33.cn/chain33/chain33/rpc/jsonclient"
	rpctypes "gitlab.33.cn/chain33/chain33/rpc/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func HashlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hashlock",
		Short: "Hashlock operation",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		HashlockLockCmd(),
		HashlockUnlockCmd(),
		HashlockSendCmd(),
	)

	return cmd
}

// 锁定
func HashlockLockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Create hashlock lock transaction",
		Run:   hashlockLockCmd,
	}
	addHashlockLockCmdFlags(cmd)
	return cmd
}

func addHashlockLockCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("secret", "s", "", "secret information")
	cmd.MarkFlagRequired("secret")
	cmd.Flags().Float64P("amount", "a", 0.0, "locking amount")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().Int64P("delay", "d", 60, "delay period (minimum 60 seconds)")
	cmd.MarkFlagRequired("delay")
	cmd.Flags().StringP("to", "t", "", "to address")
	cmd.MarkFlagRequired("to")
	cmd.Flags().StringP("return", "r", "", "return address")
	cmd.MarkFlagRequired("return")

	defaultFee := float64(types.GInt("MinFee")) / float64(types.Coin)
	cmd.Flags().Float64P("fee", "f", defaultFee, "transaction fee")
}

func hashlockLockCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	secret, _ := cmd.Flags().GetString("secret")
	toAddr, _ := cmd.Flags().GetString("to")
	returnAddr, _ := cmd.Flags().GetString("return")
	delay, _ := cmd.Flags().GetInt64("delay")
	amount, _ := cmd.Flags().GetFloat64("amount")
	fee, _ := cmd.Flags().GetFloat64("fee")

	if delay < 60 {
		fmt.Println("delay period changed to 60")
		delay = 60
	}
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4
	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := pty.HashlockLockTx{
		Secret:     secret,
		Amount:     amountInt64,
		Time:       delay,
		ToAddr:     toAddr,
		ReturnAddr: returnAddr,
		Fee:        feeInt64,
	}

	payLoad, err := json.Marshal(params)
	if err != nil {
		return
	}

	paramWithExecAction := rpctypes.CreateTxIn{
		Execer:     "hashlock",
		ActionName: "HashlockLock",
		Payload:    payLoad,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CreateTransaction", paramWithExecAction, nil)
	ctx.RunWithoutMarshal()
}

// 解锁
func HashlockUnlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock",
		Short: "Create hashlock unlock transaction",
		Run:   hashlockUnlockCmd,
	}
	addHashlockCmdFlags(cmd)
	return cmd
}

func addHashlockCmdFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("secret", "s", "", "secret information")
	cmd.MarkFlagRequired("secret")

	defaultFee := float64(types.GInt("MinFee")) / float64(types.Coin)
	cmd.Flags().Float64P("fee", "f", defaultFee, "transaction fee")
}

func hashlockUnlockCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	secret, _ := cmd.Flags().GetString("secret")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := pty.HashlockUnlockTx{
		Secret: secret,
		Fee:    feeInt64,
	}
	payLoad, err := json.Marshal(params)
	if err != nil {
		return
	}

	paramWithExecAction := rpctypes.CreateTxIn{
		Execer:     "hashlock",
		ActionName: "HashlockUnlock",
		Payload:    payLoad,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CreateTransaction", paramWithExecAction, nil)
	ctx.RunWithoutMarshal()
}

// 发送
func HashlockSendCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Create hashlock send transaction",
		Run:   hashlockSendCmd,
	}
	addHashlockCmdFlags(cmd)
	return cmd
}

func hashlockSendCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	secret, _ := cmd.Flags().GetString("secret")
	fee, _ := cmd.Flags().GetFloat64("fee")

	feeInt64 := int64(fee*types.InputPrecision) * types.Multiple1E4
	params := pty.HashlockSendTx{
		Secret: secret,
		Fee:    feeInt64,
	}
	payLoad, err := json.Marshal(params)
	if err != nil {
		return
	}

	paramWithExecAction := rpctypes.CreateTxIn{
		Execer:     "hashlock",
		ActionName: "HashlockSend",
		Payload:    payLoad,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.CreateTransaction", paramWithExecAction, nil)
	ctx.RunWithoutMarshal()
}
