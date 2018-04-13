package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func BlockHeaderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "header",
		Short: "Get block headers between [start, end]",
		Run:   blockHeader,
	}
	addBlockHeaderFlags(cmd)
	return cmd
}

func addBlockHeaderFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("start", "s", 0, "block start height")
	cmd.MarkFlagRequired("start")

	cmd.Flags().Int64P("end", "e", 0, "block end height")
	cmd.MarkFlagRequired("end")

	cmd.Flags().Bool("detail", false, "whether print block detail info")
}

func blockHeader(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	startH, _ := cmd.Flags().GetInt64("start")
	endH, _ := cmd.Flags().GetInt64("end")
	isDetail, _ := cmd.Flags().GetBool("detail")
	params := types.ReqBlocks{
		Start:    startH,
		End:      endH,
		Isdetail: isDetail,
	}
	var res jsonrpc.Headers
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetHeaders", params, &res)
	ctx.Run()
}
