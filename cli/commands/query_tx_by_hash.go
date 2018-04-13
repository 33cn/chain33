package commands

import (
	"strconv"

	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func QueryTxByHashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query_hash",
		Short: "Query transaction by hash",
		Run:   queryTxByHash,
	}
	addQueryTxByHashFlags(cmd)
	return cmd
}

func addQueryTxByHashFlags(cmd *cobra.Command) {
	// query tx by hash
	cmd.Flags().StringP("hash", "x", "", "transaction hash")
	cmd.MarkFlagRequired("hash")

	// getrawtx
	cmd.Flags().BoolP("raw_hex", "r", false, "whether output hex format")
}

func queryTxByHash(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	hash, _ := cmd.Flags().GetString("hash")
	isRawHex, _ := cmd.Flags().GetBool("raw_hex")

	if isRawHex {
		queryByHashOutputHex(rpcLaddr, hash)
	} else {
		queryByHash(rpcLaddr, hash)
	}
}

func queryByHashOutputHex(rpcAddr, txHash string) {
	params := jsonrpc.QueryParm{
		Hash: txHash,
	}
	var res string
	ctx := NewRPCCtx(rpcAddr, "Chain33.GetHexTxByHash", params, &res)
	ctx.Run()
}

func queryByHash(rpcAddr, txHash string) {
	params := jsonrpc.QueryParm{
		Hash: txHash,
	}
	var res jsonrpc.TransactionDetail
	ctx := NewRPCCtx(rpcAddr, "Chain33.QueryTransaction", params, &res)
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
