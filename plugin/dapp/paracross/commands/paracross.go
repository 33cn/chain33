package commands

import (
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/spf13/cobra"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func ParcCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "para",
		Short: "Construct para transactions",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.AddCommand(
		CreateRawAssetTransferCmd(),
		CreateRawAssetWithdrawCmd(),
		CreateRawTransferCmd(),
		CreateRawWithdrawCmd(),
		CreateRawTransferToExecCmd(),
	)
	return cmd
}

// create raw asset transfer tx
func CreateRawAssetTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset_transfer",
		Short: "Create a asset transfer to para-chain transaction",
		Run:   createAssetTransfer,
	}
	addCreateAssetTransferFlags(cmd)
	return cmd
}

func addCreateAssetTransferFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")

	cmd.Flags().StringP("title", "", "", "the title of para chain, like `p.user.guodun.`")
	cmd.MarkFlagRequired("title")

	cmd.Flags().StringP("symbol", "s", "", "default for bty, symbol for token")
}

func createAssetTransfer(cmd *cobra.Command, args []string) {
	txHex, err := createAssetTx(cmd, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create raw asset withdraw tx
func CreateRawAssetWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "asset_withdraw",
		Short: "Create a asset withdraw to para-chain transaction",
		Run:   createAssetWithdraw,
	}
	addCreateAssetWithdrawFlags(cmd)
	return cmd
}

func addCreateAssetWithdrawFlags(cmd *cobra.Command) {
	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")

	cmd.Flags().StringP("title", "", "", "the title of para chain, like `p.user.guodun.`")
	cmd.MarkFlagRequired("title")

	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().StringP("symbol", "s", "", "default for bty, symbol for token")
}

func createAssetWithdraw(cmd *cobra.Command, args []string) {
	txHex, err := createAssetTx(cmd, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

func createAssetTx(cmd *cobra.Command, isWithdraw bool) (string, error) {
	amount, _ := cmd.Flags().GetFloat64("amount")
	if amount < 0 {
		return "", types.ErrAmount
	}
	amountInt64 := int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4

	toAddr, _ := cmd.Flags().GetString("to")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")

	title, _ := cmd.Flags().GetString("title")
	if !strings.HasPrefix(title, "user.p") {
		fmt.Fprintln(os.Stderr, "title is not right, title format like `user.p.guodun.`")
		return "", types.ErrInvalidParam
	}
	execName := title + pt.ParaX

	param := types.CreateTx{
		To:          toAddr,
		Amount:      amountInt64,
		Fee:         0,
		Note:        note,
		IsWithdraw:  isWithdraw,
		IsToken:     false,
		TokenSymbol: symbol,
		ExecName:    execName,
	}
	tx, err := pt.CreateRawAssetTransferTx(&param)
	if err != nil {
		return "", err
	}

	txHex := types.Encode(tx)
	return hex.EncodeToString(txHex), nil
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
	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")

	cmd.Flags().StringP("title", "", "", "the title of para chain, like `p.user.guodun.`")
	cmd.MarkFlagRequired("title")

	cmd.Flags().StringP("symbol", "s", "", "default for bty, symbol for token")
}

func createTransfer(cmd *cobra.Command, args []string) {
	txHex, err := createTransferTx(cmd, false)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create raw transfer to exec tx
func CreateRawTransferToExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer_exec",
		Short: "Create a transfer to exec transaction",
		Run:   createTransferToExec,
	}
	addCreateTransferToExecFlags(cmd)
	return cmd
}

func addCreateTransferToExecFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("to", "t", "", "receiver exec name")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")

	cmd.Flags().StringP("title", "", "", "the title of para chain, like `p.user.guodun.`")
	cmd.MarkFlagRequired("title")

	cmd.Flags().StringP("symbol", "s", "", "default for bty, symbol for token")
}

func createTransferToExec(cmd *cobra.Command, args []string) {
	txHex, err := createTransferTx(cmd, false)
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
	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")

	cmd.Flags().StringP("title", "", "", "the title of para chain, like `p.user.guodun.`")
	cmd.MarkFlagRequired("title")

	cmd.Flags().StringP("from", "t", "", "exec name")
	cmd.MarkFlagRequired("from")

	cmd.Flags().StringP("symbol", "s", "", "default for bty, symbol for token")
}

func createWithdraw(cmd *cobra.Command, args []string) {
	txHex, err := createTransferTx(cmd, true)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

func createTransferTx(cmd *cobra.Command, isWithdraw bool) (string, error) {
	amount, _ := cmd.Flags().GetFloat64("amount")
	if amount < 0 {
		return "", types.ErrAmount
	}
	amountInt64 := int64(math.Trunc((amount+0.0000001)*1e4)) * 1e4

	toAddr, _ := cmd.Flags().GetString("to")
	note, _ := cmd.Flags().GetString("note")
	symbol, _ := cmd.Flags().GetString("symbol")

	title, _ := cmd.Flags().GetString("title")
	if !strings.HasPrefix(title, "user.p") {
		fmt.Fprintln(os.Stderr, "title is not right, title format like `user.p.guodun.`")
		return "", types.ErrInvalidParam
	}
	execName := title + pt.ParaX

	param := types.CreateTx{
		To:          toAddr,
		Amount:      amountInt64,
		Fee:         0,
		Note:        note,
		IsWithdraw:  isWithdraw,
		IsToken:     false,
		TokenSymbol: symbol,
		ExecName:    execName,
	}
	tx, err := pt.CreateRawTransferTx(&param)
	if err != nil {
		return "", err
	}

	txHex := types.Encode(tx)
	return hex.EncodeToString(txHex), nil
}
