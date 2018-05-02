package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/types"
)

func BTYCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bty",
		Short: "Construct BTY transactions",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		//TransferCmd(),
		//WithdrawFromExecCmd(),
		CreateRawTransferCmd(),
		CreateRawWithdrawCmd(),
	)

	return cmd
}

// create raw transfer tx
func CreateRawTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Create a transfer transaction",
		Run:   createTransfer,
	}
	addCreateTransferFlags(cmd)
	return cmd
}

func addCreateTransferFlags(cmd *cobra.Command) {
	//cmd.Flags().StringP("key", "k", "", "private key of sender")
	//cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.MarkFlagRequired("note")
}

func createTransfer(cmd *cobra.Command, args []string) {
	//key, _ := cmd.Flags().GetString("key")
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	txHex, err := CreateRawTx(toAddr, amount, note, false, false, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create raw withdraw tx
func CreateRawWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Create a withdraw transaction",
		Run:   createWithdraw,
	}
	addCreateWithdrawFlags(cmd)
	return cmd
}

func addCreateWithdrawFlags(cmd *cobra.Command) {
	//cmd.Flags().StringP("key", "k", "", "private key of user")
	//cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.MarkFlagRequired("note")
}

func createWithdraw(cmd *cobra.Command, args []string) {
	//key, _ := cmd.Flags().GetString("key")
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := CreateRawTx(execAddr, amount, note, true, false, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// send to address
func TransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Send coins to address",
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
}

func transfer(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	fromAddr, _ := cmd.Flags().GetString("from")
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	SendToAddress(rpcLaddr, fromAddr, toAddr, amountInt64, note, false, "", false)
}

// withdraw from executor
func WithdrawFromExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "withdraw",
		Short: "Withdraw coin from executor",
		Run:   withdraw,
	}
	addWithdrawFlags(cmd)
	return cmd
}

func addWithdrawFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "t", "", "withdraw account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func withdraw(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	SendToAddress(rpcLaddr, addr, execAddr, amountInt64, note, false, "", true)
}
