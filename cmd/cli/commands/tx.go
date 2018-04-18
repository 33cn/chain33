package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/common"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func TxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "Transaction management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		TransferCmd(),
		WithdrawFromExecCmd(),
		CreateRawTransferCmd(),
		CreateRawWithdrawCmd(),
		SendTxCmd(),
		QueryTxCmd(),
		QueryTxByAddrCmd(),
		QueryTxsByHashesCmd(),
		GetRawTxCmd(),
		DecodeTxCmd(),
	)

	return cmd
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
	sendToAddress(rpcLaddr, fromAddr, toAddr, amountInt64, note, false, "", false)
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
	execAddr, err := getExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	amountInt64 := int64(amount*types.InputPrecision) * types.Multiple1E4 //支持4位小数输入，多余的输入将被截断
	sendToAddress(rpcLaddr, addr, execAddr, amountInt64, note, false, "", true)
}

// create raw transfer tx
func CreateRawTransferCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create_transfer",
		Short: "Create a raw transfer transaction",
		Run:   createTransfer,
	}
	addCreateTransferFlags(cmd)
	return cmd
}
// TODO
func addCreateTransferFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "private key of sender")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("to", "t", "", "receiver account address")
	cmd.MarkFlagRequired("to")

	cmd.Flags().Float64P("amount", "a", 0, "transaction amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.MarkFlagRequired("note")
}

func createTransfer(cmd *cobra.Command, args []string) {
	key, _ := cmd.Flags().GetString("key")
	toAddr, _ := cmd.Flags().GetString("to")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	txHex, err := createRawTx(key, toAddr, amount, note, false, false, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// create raw withdraw tx
func CreateRawWithdrawCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create_withdraw",
		Short: "Create a raw transaction",
		Run:   createWithdraw,
	}
	addCreateWithdrawFlags(cmd)
	return cmd
}
// TODO
func addCreateWithdrawFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("key", "k", "", "private key of user")
	cmd.MarkFlagRequired("key")

	cmd.Flags().StringP("exec", "e", "", "execer withdrawn from")
	cmd.MarkFlagRequired("exec")

	cmd.Flags().Float64P("amount", "a", 0, "withdraw amount")
	cmd.MarkFlagRequired("amount")

	cmd.Flags().StringP("note", "n", "", "transaction note info")
	cmd.MarkFlagRequired("note")
}

func createWithdraw(cmd *cobra.Command, args []string) {
	key, _ := cmd.Flags().GetString("key")
	exec, _ := cmd.Flags().GetString("exec")
	amount, _ := cmd.Flags().GetFloat64("amount")
	note, _ := cmd.Flags().GetString("note")
	execAddr, err := getExecAddr(exec)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	txHex, err := createRawTx(key, execAddr, amount, note, true, false, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	fmt.Println(txHex)
}

// send raw tx
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

// get tx by address
func QueryTxByAddrCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query_addr",
		Short: "Query transaction by account address",
		Run:   queryTxByAddr,
	}
	addQueryTxByAddrFlags(cmd)
	return cmd
}

func addQueryTxByAddrFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")

	cmd.Flags().Int32P("flag", "f", 0, "transaction type(0: all txs relevant to addr, 1: addr as sender, 2: addr as receiver) (default 0)")
	cmd.Flags().Int32P("count", "c", 10, "maximum return number of transactions")
	cmd.Flags().Int32P("direction", "d", 0, "query direction from height:index(0: positive order -1:negative order) (default 0)")
	cmd.Flags().Int64P("height", "t", -1, "transaction's block height(-1: from latest txs, >=0: query from height)")
	cmd.Flags().Int64P("index", "i", 0, "query from index of tx in block height[0-100000] (default 0)")
}

func queryTxByAddr(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	flag, _ := cmd.Flags().GetInt32("flag")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("direction")
	height, _ := cmd.Flags().GetInt64("height")
	index, _ := cmd.Flags().GetInt64("index")
	params := types.ReqAddr{
		Addr:      addr,
		Flag:      flag,
		Count:     count,
		Direction: direction,
		Height:    height,
		Index:     index,
	}
	var res jsonrpc.ReplyTxInfos
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetTxByAddr", params, &res)
	ctx.Run()
}

// query tx by hash
func QueryTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query transaction by hash",
		Run:   queryTx,
	}
	addQueryTxFlags(cmd)
	return cmd
}

func addQueryTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("hash", "s", "", "transaction hash")
	cmd.MarkFlagRequired("hash")
}

func queryTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	hash, _ := cmd.Flags().GetString("hash")
	if len(hash) != 66 {
		fmt.Print(types.ErrHashNotExist.Error())
		return
	}
	params := jsonrpc.QueryParm{
		Hash: hash,
	}
	var res jsonrpc.TransactionDetail
	ctx := NewRpcCtx(rpcLaddr, "Chain33.QueryTransaction", params, &res)
	ctx.SetResultCb(parseQueryTxRes)
	ctx.Run()
}

func parseQueryTxRes(arg interface{}) (interface{}, error) {
	res := arg.(*jsonrpc.TransactionDetail)
	amountResult := strconv.FormatFloat(float64(res.Amount)/float64(types.Coin), 'f', 4, 64)
	result := TxDetailResult{
		Tx:         decodeTransaction(res.Tx),
		Receipt:    decodeLog(*(res.Receipt)),
		Proofs:     res.Proofs,
		Height:     res.Height,
		Index:      res.Index,
		Blocktime:  res.Blocktime,
		Amount:     amountResult,
		Fromaddr:   res.Fromaddr,
		ActionName: res.ActionName,
	}
	return result, nil
}

func QueryTxsByHashesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_hex",
		Short: "get transaction hex by hash",
		Run:   getTxsByHashes,
	}
	addGetTxsByHashesFlags(cmd)
	return cmd
}


// get raw transaction hex
func GetRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_hex",
		Short: "get transaction hex by hash",
		Run:   getTxHexByHash,
	}
	addGetRawTxFlags(cmd)
	return cmd
}

func addGetRawTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("hash", "s", "", "transaction hash")
	cmd.MarkFlagRequired("hash")
}

func getTxHexByHash(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	txHash, _ := cmd.Flags().GetString("hash")
	params := jsonrpc.QueryParm{
		Hash: txHash,
	}
	var res string
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetHexTxByHash", params, &res)
	ctx.Run()
}

// decode raw hex to transaction
func DecodeTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode",
		Short: "Decode a hex format transaction",
		Run:   decodeTx,
	}
	addDecodeTxFlags(cmd)
	return cmd
}

func addDecodeTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("data", "d", "", "transaction content")
	cmd.MarkFlagRequired("data")
}

func decodeTx(cmd *cobra.Command, args []string) {
	data, _ := cmd.Flags().GetString("data")
	var tx types.Transaction
	bytes, err := common.FromHex(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	err = types.Decode(bytes, &tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	res, err := jsonrpc.DecodeTx(&tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	result, err := json.MarshalIndent(res, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(result))
}