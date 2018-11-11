package commands

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/33cn/chain33/rpc/jsonclient"
	. "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
)

const (
	defaultPrivacyMixCount = 16
)

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
		CreatePub2PrivTxCmd(),
		CreatePriv2PrivTxCmd(),
		CreatePriv2PubTxCmd(),
	)
	return cmd
}

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
		CreatePub2PrivTxCmd(),
		CreatePriv2PrivTxCmd(),
		CreatePriv2PubTxCmd(),
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
	txHex, err := CreateRawTx(cmd, toAddr, amount, note, false, "", "")
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
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := CreateRawTx(cmd, execAddr, amount, note, true, "", exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create send to exec
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
	execAddr, err := GetExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := CreateRawTx(cmd, execAddr, amount, note, false, "", exec)
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

// create tx group
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
	err = group.Check(0, types.GInt("MinFee"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	newtx := group.Tx()
	grouptx := hex.EncodeToString(types.Encode(newtx))
	fmt.Println(grouptx)
}

func CreatePub2PrivTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pub2priv",
		Short: "Create a public to privacy transaction",
		Run:   createPub2PrivTx,
	}
	createPub2PrivTxFlags(cmd)
	return cmd
}

func createPub2PrivTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pubkeypair", "p", "", "public key pair")
	cmd.MarkFlagRequired("pubkeypair")
	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("tokenname", "", "BTY", "token name")
	cmd.Flags().StringP("note", "n", "", "note for transaction")
	cmd.Flags().Int64P("expire", "", 0, "transfer expire, default one hour")
	cmd.Flags().IntP("expiretype", "", 1, "0: height  1: time default is 1")
}

func createPub2PrivTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pubkeypair, _ := cmd.Flags().GetString("pubkeypair")
	amount := GetAmountValue(cmd, "amount")
	tokenname, _ := cmd.Flags().GetString("tokenname")
	note, _ := cmd.Flags().GetString("note")
	expire, _ := cmd.Flags().GetInt64("expire")
	expiretype, _ := cmd.Flags().GetInt("expiretype")
	if expiretype == 0 {
		if expire <= 0 {
			fmt.Println("Invalid expire. expire must large than 0 in expiretype==0, expire", expire)
			return
		}
	} else if expiretype == 1 {
		if expire <= 0 {
			expire = int64(time.Hour / time.Second)
		}
	} else {
		fmt.Println("Invalid expiretype", expiretype)
		return
	}

	params := types.ReqCreateTransaction{
		Tokenname:  tokenname,
		Type:       types.PrivacyTypePublic2Privacy,
		Amount:     amount,
		Note:       note,
		Pubkeypair: pubkeypair,
		Expire:     expire,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.CreateRawTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func CreatePriv2PrivTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priv2priv",
		Short: "Create a privacy to privacy transaction",
		Run:   createPriv2PrivTx,
	}
	createPriv2PrivTxFlags(cmd)
	return cmd
}

func createPriv2PrivTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("pubkeypair", "p", "", "public key pair")
	cmd.MarkFlagRequired("pubkeypair")
	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().StringP("sender", "s", "", "account address")
	cmd.MarkFlagRequired("sender")

	cmd.Flags().StringP("tokenname", "t", "BTY", "token name")
	cmd.Flags().StringP("note", "n", "", "note for transaction")
	cmd.Flags().Int64P("expire", "", 0, "transfer expire, default one hour")
	cmd.Flags().IntP("expiretype", "", 1, "0: height  1: time default is 1")
}

func createPriv2PrivTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	pubkeypair, _ := cmd.Flags().GetString("pubkeypair")
	amount := GetAmountValue(cmd, "amount")
	tokenname, _ := cmd.Flags().GetString("tokenname")
	note, _ := cmd.Flags().GetString("note")
	sender, _ := cmd.Flags().GetString("sender")
	expire, _ := cmd.Flags().GetInt64("expire")
	expiretype, _ := cmd.Flags().GetInt("expiretype")
	if expiretype == 0 {
		if expire <= 0 {
			fmt.Println("Invalid expire. expire must large than 0 in expiretype==0, expire", expire)
			return
		}
	} else if expiretype == 1 {
		if expire <= 0 {
			expire = int64(time.Hour / time.Second)
		}
	} else {
		fmt.Println("Invalid expiretype", expiretype)
		return
	}

	params := types.ReqCreateTransaction{
		Tokenname:  tokenname,
		Type:       types.PrivacyTypePrivacy2Privacy,
		Amount:     amount,
		Note:       note,
		Pubkeypair: pubkeypair,
		From:       sender,
		Mixcount:   defaultPrivacyMixCount,
		Expire:     expire,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.CreateRawTransaction", params, nil)
	ctx.RunWithoutMarshal()
}

func CreatePriv2PubTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "priv2pub",
		Short: "Create a privacy to public transaction",
		Run:   createPriv2PubTx,
	}
	createPriv2PubTxFlags(cmd)
	return cmd
}

func createPriv2PubTxFlags(cmd *cobra.Command) {
	cmd.Flags().Float64P("amount", "a", 0.0, "transfer amount, at most 4 decimal places")
	cmd.MarkFlagRequired("amount")
	cmd.Flags().StringP("from", "f", "", "from account address")
	cmd.MarkFlagRequired("from")
	cmd.Flags().StringP("to", "o", "", "to account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().StringP("tokenname", "t", "BTY", "token name")
	cmd.Flags().StringP("note", "n", "", "note for transaction")
	cmd.Flags().Int64P("expire", "", 0, "transfer expire, default one hour")
	cmd.Flags().IntP("expiretype", "", 1, "0: height  1: time default is 1")
}

func createPriv2PubTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	amount := GetAmountValue(cmd, "amount")
	tokenname, _ := cmd.Flags().GetString("tokenname")
	from, _ := cmd.Flags().GetString("from")
	to, _ := cmd.Flags().GetString("to")
	note, _ := cmd.Flags().GetString("note")
	expire, _ := cmd.Flags().GetInt64("expire")
	expiretype, _ := cmd.Flags().GetInt("expiretype")
	if expiretype == 0 {
		if expire <= 0 {
			fmt.Println("Invalid expire. expire must large than 0 in expiretype==0, expire", expire)
			return
		}
	} else if expiretype == 1 {
		if expire <= 0 {
			expire = int64(time.Hour / time.Second)
		}
	} else {
		fmt.Println("Invalid expiretype", expiretype)
		return
	}

	params := types.ReqCreateTransaction{
		Tokenname: tokenname,
		Type:      types.PrivacyTypePrivacy2Public,
		Amount:    amount,
		Note:      note,
		From:      from,
		To:        to,
		Mixcount:  defaultPrivacyMixCount,
		Expire:    expire,
	}
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "privacy.CreateRawTransaction", params, nil)
	ctx.RunWithoutMarshal()
}
