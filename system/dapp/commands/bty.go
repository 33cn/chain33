// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// BTYCmd bty command
func BTYCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bty",
		Short: "Construct BTY transactions",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		CreateRawTransferCmd(),
		CreateRawWithdrawCmd(),
		CreateRawSendToExecCmd(),
		CreateTxGroupCmd(),
	)
	return cmd
}

// CoinsCmd coins command func
func CoinsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coins",
		Short: "Construct system coins transactions",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		CreateRawTransferCmd(),
		CreateRawWithdrawCmd(),
		CreateRawSendToExecCmd(),
		CreateTxGroupCmd(),
	)
	return cmd
}

// CreateRawTransferCmd create raw transfer tx
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
	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func createTransfer(cmd *cobra.Command, args []string) {
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	txHex, err := commandtypes.CreateRawTx(cmd, toAddr, amount, note, false, "", "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// CreateRawWithdrawCmd  create raw withdraw tx
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
	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func createWithdraw(cmd *cobra.Command, args []string) {
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := commandtypes.GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := commandtypes.CreateRawTx(cmd, execAddr, amount, note, true, "", exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// CreateRawSendToExecCmd  create send to exec
func CreateRawSendToExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send_exec",
		Short: "Create a send to executor transaction",
		Run:   sendToExec,
	}
	addCreateRawSendToExecFlags(cmd)
	return cmd
}

func addCreateRawSendToExecFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("exec", "e", "", "executor to be sent to")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "send amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
}

func sendToExec(cmd *cobra.Command, args []string) {
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := commandtypes.GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := commandtypes.CreateRawTx(cmd, execAddr, amount, note, false, "", exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// TransferCmd send to address
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
	commandtypes.SendToAddress(rpcLaddr, fromAddr, toAddr, amountInt64, note, false, "", false)
}

// WithdrawFromExecCmd withdraw from executor
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
	execAddr, err := commandtypes.GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	commandtypes.SendToAddress(rpcLaddr, addr, execAddr, amountInt64, note, false, "", true)
}

// CreateTxGroupCmd create tx group
func CreateTxGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "txgroup",
		Short: "Create a transaction group",
		Run:   createTxGroup,
	}
	addCreateTxGroupFlags(cmd)
	return cmd
}

func addCreateTxGroupFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("txs", "t", "", "transactions in hex, separated by space")
	cmd.Flags().StringP("file", "f", "", "name of file which contains hex style transactions, separated by new line")
}

func createTxGroup(cmd *cobra.Command, args []string) {
	txs, _ := cmd.Flags().GetString("txs")
	file, _ := cmd.Flags().GetString("file")
	var txsArr []string
	if txs != "" {
		txsArr = strings.Split(txs, " ")
	} else if file != "" {
		f, err := os.Open(file)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer f.Close()
		rd := bufio.NewReader(f)
		i := 1
		for {
			line, _, err := rd.ReadLine()
			if err != nil && err != io.EOF {
				fmt.Fprintln(os.Stderr, err)
				return
			}
			if err == io.EOF {
				break
			}
			cSet := " 	" // space and tab
			lineStr := strings.Trim(string(line), cSet)
			if lineStr == "" {
				continue
			}
			fmt.Printf("tx %d: %s", i, lineStr+"\n")
			txsArr = append(txsArr, lineStr)
			i++
		}
	} else {
		fmt.Println("please input -t or -f; else, input -h to see help")
		return
	}
	var transactions []*types.Transaction
	for _, t := range txsArr {
		txByte, err := hex.DecodeString(t)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		var transaction types.Transaction
		types.Decode(txByte, &transaction)
		transactions = append(transactions, &transaction)
	}
	group, err := types.CreateTxGroup(transactions)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	err = group.Check(0, types.GInt("MinFee"), types.GInt("MaxFee"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	newtx := group.Tx()
	grouptx := hex.EncodeToString(types.Encode(newtx))
	fmt.Println(grouptx)
}
