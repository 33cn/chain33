package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/client"
)

func MempoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mempool",
		Short: "Mempool management",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GetMempoolCmd(),
		GetLastMempoolCmd(),
	)

	return cmd
}

// get mempool
func GetMempoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List mempool txs",
		Run:   listMempoolTxs,
	}
	return cmd
}

func listMempoolTxs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res jsonrpc.ReplyTxList
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetMempool", nil, &res)
	ctx.SetResultCb(parseListMempoolTxsRes)
	ctx.Run()
}

func parseListMempoolTxsRes(arg interface{}) (interface{}, error) {
	res := arg.(*jsonrpc.ReplyTxList)
	var result TxListResult
	for _, v := range res.Txs {
		result.Txs = append(result.Txs, decodeTransaction(v))
	}
	return result, nil
}

// get last 10 txs of mempool
func GetLastMempoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "last_txs",
		Short: "Get latest mempool txs",
		Run:   lastMempoolTxs,
	}
	return cmd
}

func lastMempoolTxs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res jsonrpc.ReplyTxList
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetLastMemPool", nil, &res)
	ctx.SetResultCb(parselastMempoolTxsRes)
	ctx.Run()
}

func parselastMempoolTxsRes(arg interface{}) (interface{}, error) {
	res := arg.(*jsonrpc.ReplyTxList)
	var result TxListResult
	for _, v := range res.Txs {
		result.Txs = append(result.Txs, decodeTransaction(v))
	}
	return result, nil
}
