// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package commands

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/33cn/chain33/blockchain"
	"github.com/33cn/chain33/rpc/jsonclient"
	rpctypes "github.com/33cn/chain33/rpc/types"
	commandtypes "github.com/33cn/chain33/system/dapp/commands/types"
	"github.com/33cn/chain33/types"
	"github.com/spf13/cobra"
)

// BlockCmd block command
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
		AddPushSubscribeCmd(),
		ListPushesCmd(),
		GetPushSeqLastNumCmd(),
	)

	return cmd
}

// GetBlocksCmd get blocks between start and end
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetBlocks", params, &res)
	ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

func parseBlockDetail(res interface{}) (interface{}, error) {
	var result commandtypes.BlockDetailsResult
	for _, vItem := range res.(*rpctypes.BlockDetails).Items {
		b := &commandtypes.BlockResult{
			Version:    vItem.Block.Version,
			ParentHash: vItem.Block.ParentHash,
			TxHash:     vItem.Block.TxHash,
			StateHash:  vItem.Block.StateHash,
			Height:     vItem.Block.Height,
			BlockTime:  vItem.Block.BlockTime,
		}
		for _, vTx := range vItem.Block.Txs {
			b.Txs = append(b.Txs, commandtypes.DecodeTransaction(vTx))
		}
		bd := &commandtypes.BlockDetailResult{Block: b, Receipts: vItem.Receipts}
		result.Items = append(result.Items, bd)
	}
	return result, nil
}

// GetBlockHashCmd get hash of a block
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetBlockHash", params, &res)
	ctx.Run()
}

// GetBlockOverviewCmd get overview of a block
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetBlockOverview", params, &res)
	ctx.Run()
}

// GetHeadersCmd get block headers between start and end
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetHeaders", params, &res)
	ctx.Run()
}

// GetLastHeaderCmd get information of latest header
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetLastHeader", nil, &res)
	ctx.Run()
}

// GetLastBlockSequenceCmd get latest Sequence
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetLastBlockSequence", nil, &res)
	ctx.Run()
}

// GetBlockSequencesCmd  get block Sequences
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
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetBlockSequences", params, &res)
	//ctx.SetResultCb(parseBlockDetail)
	ctx.Run()
}

func blockSequencesCmdFlags(cmd *cobra.Command) {
	cmd.Flags().Int64P("start", "s", 0, "block start sequence")
	cmd.MarkFlagRequired("start")

	cmd.Flags().Int64P("end", "e", 0, "block end sequence")
	cmd.MarkFlagRequired("end")
}

// GetBlockByHashsCmd get Block Details By block Hashs
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
	hashesArr := strings.Fields(hashes)

	params := rpctypes.ReqHashes{
		Hashes: hashesArr,
	}

	var res rpctypes.BlockDetails
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetBlockByHashes", params, &res)
	//ctx.SetResultCb(parseQueryTxsByHashesRes)
	ctx.Run()
}

// AddPushSubscribeCmd add block sequence call back
func AddPushSubscribeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add_push",
		Short: "add push for block or tx receipt",
		Run:   addPushSubscribe,
	}
	addPushSubscribeFlags(cmd)
	return cmd
}

func addPushSubscribeFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "call back name")
	cmd.MarkFlagRequired("name")

	cmd.Flags().StringP("url", "u", "", "call back URL")
	cmd.MarkFlagRequired("url")

	cmd.Flags().StringP("encode", "e", "", "data encode type,json or proto buff")
	cmd.MarkFlagRequired("encode")

	cmd.Flags().StringP("isheader", "i", "f", "push header or block (0/f/false for block; 1/t/true for header)")

	cmd.Flags().Int64P("lastSequence", "", 0, "lastSequence")
	cmd.Flags().Int64P("lastHeight", "", 0, "lastHeight")
	cmd.Flags().StringP("lastBlockHash", "", "", "lastBlockHash")
}

func addPushSubscribe(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	name, _ := cmd.Flags().GetString("name")
	url, _ := cmd.Flags().GetString("url")
	encode, _ := cmd.Flags().GetString("encode")

	isHeaderStr, _ := cmd.Flags().GetString("isheader")
	isHeader, err := strconv.ParseBool(isHeaderStr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	lastSeq, _ := cmd.Flags().GetInt64("lastSequence")
	lastHeight, _ := cmd.Flags().GetInt64("lastHeight")
	lastBlockHash, _ := cmd.Flags().GetString("lastBlockHash")
	if lastSeq != 0 || lastHeight != 0 || lastBlockHash != "" {
		if lastSeq == 0 || lastHeight == 0 || lastBlockHash == "" {
			fmt.Println("lastSequence, lastHeight, lastBlockHash need at the same time")
			return
		}
	}
	pushType := blockchain.PushBlock
	if isHeader {
		pushType = blockchain.PushBlockHeader
	}

	params := types.PushSubscribeReq{
		Name:          name,
		URL:           url,
		Encode:        encode,
		LastSequence:  lastSeq,
		LastHeight:    lastHeight,
		LastBlockHash: lastBlockHash,
		Type:          pushType,
	}

	var res types.ReplySubscribePush
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.AddPushSubscribe", params, &res)
	ctx.Run()
}

// ListPushesCmd list block sequence call back
func ListPushesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list_pushes",
		Short: "list pushes",
		Run:   listPushes,
	}
	return cmd
}

func listPushes(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")

	var res types.PushSubscribes
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.ListPushes", nil, &res)
	ctx.Run()
}

// GetPushSeqLastNumCmd Get Seq Call Back Last Num
func GetPushSeqLastNumCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "last_push",
		Short: "show the sequence number of last push",
		Run:   getPushSeqLastNumCmd,
	}
	getPushSeqLastNumFlags(cmd)
	return cmd
}

func getPushSeqLastNumFlags(cmd *cobra.Command) {
	cmd.Flags().StringP("name", "n", "", "call back name")
	cmd.MarkFlagRequired("name")
}

func getPushSeqLastNumCmd(cmd *cobra.Command, args []string) {
	rpcLaddr, _ := cmd.Flags().GetString("rpc_laddr")
	name, _ := cmd.Flags().GetString("name")

	params := types.ReqString{
		Data: name,
	}

	var res types.Int64
	ctx := jsonclient.NewRPCCtx(rpcLaddr, "Chain33.GetPushSeqLastNum", params, &res)
	ctx.Run()
}
