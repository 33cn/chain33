package commands

import (
	"github.com/spf13/cobra"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	. "github.com/33cn/chain33/system/dapp/commands/types"
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
	var res rpctypes.ReplyTxList
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetMempool", nil, &res)
	ctx.SetResultCb(parseListMempoolTxsRes)
	ctx.Run()
}

func parseListMempoolTxsRes(arg interface{}) (interface{}, error) {
	res := arg.(*rpctypes.ReplyTxList)
	var result TxListResult
	for _, v := range res.Txs {
		result.Txs = append(result.Txs, DecodeTransaction(v))
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
	var res rpctypes.ReplyTxList
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetLastMemPool", nil, &res)
	ctx.SetResultCb(parselastMempoolTxsRes)
	ctx.Run()
}

func parselastMempoolTxsRes(arg interface{}) (interface{}, error) {
	res := arg.(*rpctypes.ReplyTxList)
	var result TxListResult
	for _, v := range res.Txs {
		result.Txs = append(result.Txs, DecodeTransaction(v))
	}
	return result, nil
}
