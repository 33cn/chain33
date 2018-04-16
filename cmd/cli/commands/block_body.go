package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
)

func BlockBodyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "body",
		Short: "Get block headers between [start, end]",
		Run:   blockBodyCmd,
	}
	addBlockBodyCmdFlags(cmd)
	return cmd
}

func addBlockBodyCmdFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("start", "s", 0, "block start height")
	cmd.MarkFlagRequired("start")

	cmd.Flags().Int64P("end", "e", 0, "block end height")
	cmd.MarkFlagRequired("end")

	cmd.Flags().Bool("detail", false, "whether print block detail info")
}

func blockBodyCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	startH, _ := cmd.Flags().GetInt64("start")
	endH, _ := cmd.Flags().GetInt64("end")
	isDetail, _ := cmd.Flags().GetBool("detail")
	params := jsonrpc.BlockParam{
		Start:    startH,
		End:      endH,
		Isdetail: isDetail,
	}
	var res jsonrpc.BlockDetails
	ctx := NewRPCCtx(rpcLaddr, "Chain33.GetBlocks", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

func parseBlockDetail(res interface{}) (interface{}, error) {
	var result BlockDetailsResult

	for _, vItem := range res.(*jsonrpc.BlockDetails).Items {
		b := &BlockResult{
			Version:    vItem.Block.Version,
			ParentHash: vItem.Block.ParentHash,
			TxHash:     vItem.Block.TxHash,
			StateHash:  vItem.Block.StateHash,
			Height:     vItem.Block.Height,
			BlockTime:  vItem.Block.BlockTime,
		}
		for _, vTx := range vItem.Block.Txs {
			b.Txs = append(b.Txs, decodeTransaction(vTx))
		}
		var rpt []*ReceiptData
		for _, vR := range vItem.Receipts {
			rpt = append(rpt, decodeLog(*vR))
		}
		bd := &BlockDetailResult{Block: b, Receipts: rpt}
		result.Items = append(result.Items, bd)
	}

	return result, nil
}
