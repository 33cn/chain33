package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	. "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
)

func BlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "block",
		Short: "Get block header or body info",
		Args:  cobra.MinimumNArgs(1),
	}

	cmd.AddCommand(
		GetBlocksCmd(),
		GetBlockHashCmd(),
		GetBlockOverviewCmd(),
		GetHeadersCmd(),
		GetLastHeaderCmd(),

		GetBlockByHashsCmd(),
		GetBlockSequencesCmd(),
		GetLastBlockSequenceCmd(),
	)

	return cmd
}

// get blocks between start and end
func GetBlocksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get blocks between [start, end]",
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

	cmd.Flags().StringP("detail", "d", "f", "whether print block detail info (0/f/false for No; 1/t/true for Yes)")
}

func blockBodyCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	startH, _ := cmd.Flags().GetInt64("start")
	endH, _ := cmd.Flags().GetInt64("end")
	isDetail, _ := cmd.Flags().GetString("detail")
	detailBool, err := strconv.ParseBool(isDetail)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := rpctypes.BlockParam{
		Start:    startH,
		End:      endH,
		Isdetail: detailBool,
	}
	var res rpctypes.BlockDetails
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetBlocks", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

func parseBlockDetail(res interface{}) (interface{}, error) {
	var result BlockDetailsResult
	for _, vItem := range res.(*rpctypes.BlockDetails).Items {
		b := &BlockResult{
			Version:    vItem.Block.Version,
			ParentHash: vItem.Block.ParentHash,
			TxHash:     vItem.Block.TxHash,
			StateHash:  vItem.Block.StateHash,
			Height:     vItem.Block.Height,
			BlockTime:  vItem.Block.BlockTime,
		}
		for _, vTx := range vItem.Block.Txs {
			b.Txs = append(b.Txs, DecodeTransaction(vTx))
		}
		bd := &BlockDetailResult{Block: b, Receipts: vItem.Receipts}
		result.Items = append(result.Items, bd)
	}
	return result, nil
}

// get hash of a block
func GetBlockHashCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hash",
		Short: "Get hash of block at height",
		Run:   blockHeightHash,
	}
	addBlockHashFlags(cmd)
	return cmd
}

func addBlockHashFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("height", "t", 0, "block height")
	cmd.MarkFlagRequired("height")
}

func blockHeightHash(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	height, _ := cmd.Flags().GetInt64("height")
	params := types.ReqInt{
		Height: height,
	}
	var res rpctypes.ReplyHash
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetBlockHash", params, &res)
	ctx.Run()
}

// get overview of a block
func GetBlockOverviewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "view",
		Short: "View block info by block hash",
		Run:   blockViewByHash,
	}
	addBlockViewFlags(cmd)
	return cmd
}

func addBlockViewFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("hash", "s", "", "block hash at height")
	cmd.MarkFlagRequired("hash")
}

func blockViewByHash(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	blockHash, _ := cmd.Flags().GetString("hash")
	params := rpctypes.QueryParm{
		Hash: blockHash,
	}
	var res rpctypes.BlockOverview
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetBlockOverview", params, &res)
	ctx.Run()
}

// get block headers between start and end
func GetHeadersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "headers",
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

	cmd.Flags().StringP("detail", "d", "f", "whether print header detail info (0/f/false for No; 1/t/true for Yes)")
}

func blockHeader(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	startH, _ := cmd.Flags().GetInt64("start")
	endH, _ := cmd.Flags().GetInt64("end")
	isDetail, _ := cmd.Flags().GetString("detail")
	detailBool, err := strconv.ParseBool(isDetail)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}
	params := types.ReqBlocks{
		Start:    startH,
		End:      endH,
		IsDetail: detailBool,
	}
	var res rpctypes.Headers
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetHeaders", params, &res)
	ctx.Run()
}

// get information of latest header
func GetLastHeaderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "last_header",
		Short: "View last block header",
		Run:   lastHeader,
	}
	return cmd
}

func lastHeader(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res rpctypes.Header
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetLastHeader", nil, &res)
	ctx.Run()
}

//
// get latest Sequence
func GetLastBlockSequenceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "last_sequence",
		Short: "View last block sequence",
		Run:   lastSequence,
	}
	return cmd
}

func lastSequence(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	var res int64
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetLastBlockSequence", nil, &res)
	ctx.Run()
}

// get block Sequences
func GetBlockSequencesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sequences",
		Short: "Get block sequences between [start, end]",
		Run:   getsequences,
	}
	blockSequencesCmdFlags(cmd)
	return cmd
}

func getsequences(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	startH, _ := cmd.Flags().GetInt64("start")
	endH, _ := cmd.Flags().GetInt64("end")

	params := rpctypes.BlockParam{
		Start:    startH,
		End:      endH,
		Isdetail: false,
	}
	var res rpctypes.ReplyBlkSeqs
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetBlockSequences", params, &res)
	//ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

func blockSequencesCmdFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("start", "s", 0, "block start sequence")
	cmd.MarkFlagRequired("start")

	cmd.Flags().Int64P("end", "e", 0, "block end sequence")
	cmd.MarkFlagRequired("end")
}

// get Block Details By block Hashs
func GetBlockByHashsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "query_hashs",
		Short: "Query block by hashs",
		Run:   getblockbyhashs,
	}
	addBlockByHashsFlags(cmd)
	return cmd
}

func addBlockByHashsFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("hashes", "s", "", "block hash(es), separated by space")
	cmd.MarkFlagRequired("hashes")
}

func getblockbyhashs(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	hashes, _ := cmd.Flags().GetString("hashes")
	hashesArr := strings.Split(hashes, " ")
	params := rpctypes.ReqHashes{
		Hashes: hashesArr,
	}

	var res types.BlockDetails
	ctx := jsonclient.NewRpcCtx(rpcLaddr, "Chain33.GetBlockByHashes", params, &res)
	//ctx.SetResultCb(parseQueryTxsByHashesRes)
	ctx.Run()
}
