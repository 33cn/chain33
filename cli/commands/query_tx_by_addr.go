package commands

import (
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
	"github.com/spf13/cobra"
)

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
	cmd.Flags().Int32P("dir", "d", 0, "query direction from height:index(0: positive order -1:negative order) (default 0)")
	cmd.Flags().Int64("height", -1, "transaction's block height(-1: from latest txs, >=0: query from height)")
	cmd.Flags().Int64P("index", "i", 0, "query from index of tx in block height[0-100000] (default 0)")
}

func queryTxByAddr(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	addr, _ := cmd.Flags().GetString("addr")
	flag, _ := cmd.Flags().GetInt32("flag")
	count, _ := cmd.Flags().GetInt32("count")
	direction, _ := cmd.Flags().GetInt32("dir")
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
