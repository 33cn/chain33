// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"strings"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// TxCmd transaction command
func TxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tx",
		Short: "Transaction management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		QueryTxCmd(),
		QueryTxByAddrCmd(),
		QueryTxsByHashesCmd(),
		GetRawTxCmd(),
		DecodeTxCmd(),
		GetAddrOverviewCmd(),
		ReWriteRawTxCmd(),
	)

	return cmd
}

// QueryTxByAddrCmd get tx by address
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
	var res rpctypes.ReplyTxInfos
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetTxByAddr", params, &res)
	ctx.Run()
}

// QueryTxCmd  query tx by hash
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
	params := rpctypes.QueryParm{
		Hash: hash,
	}
	var res rpctypes.TransactionDetail
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.QueryTransaction", params, &res)
	ctx.SetResultCb(parseQueryTxRes)
	ctx.Run()
}

func parseQueryTxRes(arg interface{}) (interface{}, error) {
	res := arg.(*rpctypes.TransactionDetail)
	amountResult := strconv.FormatFloat(float64(res.Amount)/float64(types.Coin), 'f', 4, 64)
	result := commandtypes.TxDetailResult{
		Tx:         commandtypes.DecodeTransaction(res.Tx),
		Receipt:    res.Receipt,
		Proofs:     res.Proofs,
		Height:     res.Height,
		Index:      res.Index,
		Blocktime:  res.Blocktime,
		Amount:     amountResult,
		Fromaddr:   res.Fromaddr,
		ActionName: res.ActionName,
		Assets:     res.Assets,
	}
	return result, nil
}

// QueryTxsByHashesCmd  get transactions by hashes
func QueryTxsByHashesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query_hash",
		Short: "Get transactions by hashes",
		Run:   getTxsByHashes,
	}
	addGetTxsByHashesFlags(cmd)
	return cmd
}

func addGetTxsByHashesFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("hashes", "s", "", "transaction hash(es), separated by space")
	cmd.MarkFlagRequired("hashes")
}

func getTxsByHashes(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	hashes, _ := cmd.Flags().GetString("hashes")
	hashesArr := strings.Split(hashes, " ")
	params := rpctypes.ReqHashes{
		Hashes: hashesArr,
	}

	var res rpctypes.TransactionDetails
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetTxByHashes", params, &res)
	ctx.SetResultCb(parseQueryTxsByHashesRes)
	ctx.Run()
}

func parseQueryTxsByHashesRes(arg interface{}) (interface{}, error) {
	var result commandtypes.TxDetailsResult
	for _, v := range arg.(*rpctypes.TransactionDetails).Txs {
		if v == nil {
			result.Txs = append(result.Txs, nil)
			continue
		}
		amountResult := strconv.FormatFloat(float64(v.Amount)/float64(types.Coin), 'f', 4, 64)
		td := commandtypes.TxDetailResult{
			Tx:         commandtypes.DecodeTransaction(v.Tx),
			Receipt:    v.Receipt,
			Proofs:     v.Proofs,
			Height:     v.Height,
			Index:      v.Index,
			Blocktime:  v.Blocktime,
			Amount:     amountResult,
			Fromaddr:   v.Fromaddr,
			ActionName: v.ActionName,
			Assets:     v.Assets,
		}
		result.Txs = append(result.Txs, &td)
	}
	return result, nil
}

// GetRawTxCmd get raw transaction hex
func GetRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get_hex",
		Short: "Get transaction hex by hash",
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
	params := rpctypes.QueryParm{
		Hash: txHash,
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetHexTxByHash", params, nil)
	ctx.RunWithoutMarshal()
}

// DecodeTxCmd decode raw hex to transaction
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

	res, err := rpctypes.DecodeTx(&tx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	txResult := commandtypes.DecodeTransaction(res)

	result, err := json.MarshalIndent(txResult, "", "    ")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println(string(result))
}

// GetAddrOverviewCmd get overview of an address
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetAddrOverview", params, &res)
	ctx.SetResultCb(parseAddrOverview)
	ctx.Run()
}

func parseAddrOverview(view interface{}) (interface{}, error) {
	res := view.(*types.AddrOverview)
	balance := strconv.FormatFloat(float64(res.GetBalance())/float64(types.Coin), 'f', 4, 64)
	receiver := strconv.FormatFloat(float64(res.GetReciver())/float64(types.Coin), 'f', 4, 64)
	addrOverview := &commandtypes.AddrOverviewResult{
		Balance:  balance,
		Receiver: receiver,
		TxCount:  res.GetTxCount(),
	}
	return addrOverview, nil
}

// ReWriteRawTxCmd re-write raw transaction hex
func ReWriteRawTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rewrite",
		Short: "rewrite transaction parameters",
		Run:   reWriteRawTx,
	}
	addReWriteRawTxFlags(cmd)
	return cmd
}

func addReWriteRawTxFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("tx", "s", "", "transaction hex")
	cmd.MarkFlagRequired("tx")

	cmd.Flags().StringP("to", "t", "", "to addr (optional)")
	cmd.Flags().Float64P("fee", "f", 0, "transaction fee (optional)")
	cmd.Flags().StringP("expire", "e", "120s", "expire time (optional)")
}

func reWriteRawTx(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	txHash, _ := cmd.Flags().GetString("tx")
	to, _ := cmd.Flags().GetString("to")
	fee, _ := cmd.Flags().GetFloat64("fee")
	expire, _ := cmd.Flags().GetString("expire")
	expire, err := commandtypes.CheckExpireOpt(expire)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	feeInt64 := int64(fee * 1e4)

	params := rpctypes.ReWriteRawTx{
		Tx:     txHash,
		To:     to,
		Fee:    feeInt64 * 1e4,
		Expire: expire,
	}

	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.ReWriteRawTx", params, nil)
	ctx.RunWithoutMarshal()
}
