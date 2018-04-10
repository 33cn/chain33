package commands

import (
	"github.com/spf13/cobra"
	jsonrpc "gitlab.33.cn/chain33/chain33/rpc"
	"gitlab.33.cn/chain33/chain33/types"
)

func BlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Get block header or body info",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		LastHeaderCmd(),
		BlockHeightHashCmd(),
		BlockViewCmd(),
		BlockHeaderCmd(),
		BlockBodyCmd(),
	)

	return cmd
}

// last_header
func LastHeaderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "last_header",
		Short: "View last block header",
		Run:   lastHeader,
	}
	return cmd
}

func lastHeader(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res jsonrpc.Header
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetLastHeader", nil, &res)
	ctx.Run()
}

// hash
func BlockHeightHashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hash",
		Short: "Get hash of block at height",
		Run:   blockHeightHash,
	}
	addBlockHashFlags(cmd)
	return cmd
}

func addBlockHashFlags(cmd *cobra.Command) {
	cmd.Flags().Int64("height", 0, "block height")
	cmd.MarkFlagRequired("height")
}

func blockHeightHash(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	height, _ := cmd.Flags().GetInt64("height")
	params := types.ReqInt{
		Height: height,
	}
	var res jsonrpc.ReplyHash
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetBlockHash", params, &res)
	ctx.Run()
}

// view
func BlockViewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "View block info by block hash",
		Run:   blockViewByHash,
	}
	addBlockViewFlags(cmd)
	return cmd
}

func addBlockViewFlags(cmd *cobra.Command) {
	cmd.Flags().String("hash", "", "block hash at height")
	cmd.MarkFlagRequired("hash")
}

func blockViewByHash(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	blockHash, _ := cmd.Flags().GetString("hash")
	params := jsonrpc.QueryParm{
		Hash: blockHash,
	}
	var res jsonrpc.BlockOverview
	ctx := NewRpcCtx(rpcLaddr, "Chain33.GetBlockOverview", params, &res)
	ctx.Run()
}
