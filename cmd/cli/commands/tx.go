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
		SendTxCmd(),
		QueryTxCmd(),
		QueryTxByAddrCmd(),
		QueryTxsByHashesCmd(),
		GetRawTxCmd(),
		DecodeTxCmd(),
		GetAddrOverviewCmd(),
	)

	return cmd
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
		Use:   "query_hash",
		Short: "get transactions by hashes",
		Run:   getTxsByHashes,
	}
	addGetTxsByHashesFlags(cmd)
	return cmd
}

func addGetTxsByHashesFlags(cmd *cobra.Command) {
	emptySlice := []string{""}
	cmd.Flags().StringSliceP("hashes", "s", emptySlice, "transaction hash")
	cmd.MarkFlagRequired("hashes")
}

func getTxsByHashes(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	hashes, _ := cmd.Flags().GetStringSlice("hashes")
	params := jsonrpc.ReqHashes{
		Hashes: hashes,
	}

	var res jsonrpc.TransactionDetails
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetTxByHashes", params, &res)
	ctx.SetResultCb(parseQueryTxsByHashesRes)
	ctx.Run()
}

func parseQueryTxsByHashesRes(arg interface{}) (interface{}, error) {
	var result TxDetailsResult
	for _, v := range arg.(*jsonrpc.TransactionDetails).Txs {
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(types.Coin), 'f', 4, 64)
		td := TxDetailResult{
			Tx:         decodeTransaction(v.Tx),
			Receipt:    decodeLog(*(v.Receipt)),
			Proofs:     v.Proofs,
			Height:     v.Height,
			Index:      v.Index,
			Blocktime:  v.Blocktime,
			Amount:     amountResult,
			Fromaddr:   v.Fromaddr,
			ActionName: v.ActionName,
		}
		result.Txs = append(result.Txs, &td)
	}
	return result, nil
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

// get overview of an address
func GetAddrOverviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addr_overview",
		Short: "View transactions of address",
		Run:   viewAddress,
	}
	addAddrViewFlags(cmd)
	return cmd
}

func addAddrViewFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("addr", "a", "", "account address")
	cmd.MarkFlagRequired("addr")
}

func viewAddress(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	params := types.ReqAddr{
		Addr: addr,
	}

	var res types.AddrOverview
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetAddrOverview", params, &res)
	ctx.SetResultCb(parseAddrOverview)
	ctx.Run()
}

func parseAddrOverview(view interface{}) (interface{}, error) {
	res := view.(*types.AddrOverview)
	balance := strconv.FormatFloat(float64(res.GetBalance())/float64(types.Coin), 'f', 4, 64)
	receiver := strconv.FormatFloat(float64(res.GetReciver())/float64(types.Coin), 'f', 4, 64)
	addrOverview := &AddrOverviewResult{
		Balance:  balance,
		Receiver: receiver,
		TxCount:  res.GetTxCount(),
	}
	return addrOverview, nil
}
