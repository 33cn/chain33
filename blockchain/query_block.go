// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
)

//GetBlockByHashes 通过blockhash 获取对应的block信息
//从数据库获取区块不能太多，防止内存异常。一次最多获取100M区块数据从数据库
func (chain *BlockChain) GetBlockByHashes(hashes [][]byte) (respblocks *types.BlockDetails, err error) {
	if int64(len(hashes)) > types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	var blocks types.BlockDetails
	size := 0
	for _, hash := range hashes {
		block, err := chain.LoadBlockByHash(hash)
		if err == nil && block != nil {
			size += block.Size()
			if size > types.MaxBlockSizePerTime {
				chainlog.Error("GetBlockByHashes:overflow", "MaxBlockSizePerTime", types.MaxBlockSizePerTime)
				return &blocks, nil
			}
			blocks.Items = append(blocks.Items, block)
		} else {
			blocks.Items = append(blocks.Items, new(types.BlockDetail))
		}
	}
	return &blocks, nil
}

//ProcGetBlockHash 通过blockheight 获取blockhash
func (chain *BlockChain) ProcGetBlockHash(height *types.ReqInt) (*types.ReplyHash, error) {
	if height == nil || 0 > height.GetHeight() {
		chainlog.Error("ProcGetBlockHash input err!")
		return nil, types.ErrInvalidParam
	}
	CurHeight := chain.GetBlockHeight()
	if height.GetHeight() > CurHeight {
		chainlog.Error("ProcGetBlockHash input height err!")
		return nil, types.ErrInvalidParam
	}
	var ReplyHash types.ReplyHash
	var err error
	ReplyHash.Hash, err = chain.blockStore.GetBlockHashByHeight(height.GetHeight())
	if err != nil {
		return nil, err
	}
	return &ReplyHash, nil
}

//ProcGetBlockOverview 返回值
// type  BlockOverview {
//	Header head = 1;
//	int64  txCount = 2;
//	repeated bytes txHashes = 3;}
//获取BlockOverview
func (chain *BlockChain) ProcGetBlockOverview(ReqHash *types.ReqHash) (*types.BlockOverview, error) {
	if ReqHash == nil {
		chainlog.Error("ProcGetBlockOverview input err!")
		return nil, types.ErrInvalidParam
	}
	var blockOverview types.BlockOverview
	//通过height获取block
	block, err := chain.LoadBlockByHash(ReqHash.Hash)
	if err != nil || block == nil {
		chainlog.Error("ProcGetBlockOverview", "GetBlock err ", err)
		return nil, err
	}

	//获取header的信息从block中
	var header types.Header
	header.Version = block.Block.Version
	header.ParentHash = block.Block.ParentHash
	header.TxHash = block.Block.TxHash
	header.StateHash = block.Block.StateHash
	header.BlockTime = block.Block.BlockTime
	header.Height = block.Block.Height
	header.Hash = block.Block.Hash(chain.client.GetConfig())
	header.TxCount = int64(len(block.Block.GetTxs()))
	header.Difficulty = block.Block.Difficulty
	header.Signature = block.Block.Signature

	blockOverview.Head = &header

	blockOverview.TxCount = int64(len(block.Block.GetTxs()))

	txhashs := make([][]byte, blockOverview.TxCount)
	for index, tx := range block.Block.Txs {
		txhashs[index] = tx.Hash()
	}
	blockOverview.TxHashes = txhashs
	chainlog.Debug("ProcGetBlockOverview", "blockOverview:", blockOverview.String())
	return &blockOverview, nil
}

//ProcGetLastBlockMsg 获取最新区块信息
func (chain *BlockChain) ProcGetLastBlockMsg() (respblock *types.Block, err error) {
	block := chain.blockStore.LastBlock()
	return block, nil
}

//ProcGetBlockByHashMsg 获取最新区块hash
func (chain *BlockChain) ProcGetBlockByHashMsg(hash []byte) (respblock *types.BlockDetail, err error) {
	blockdetail, err := chain.LoadBlockByHash(hash)
	if err != nil {
		return nil, err
	}
	return blockdetail, nil
}

//ProcGetHeadersMsg 返回值
//type Header struct {
//	Version    int64
//	ParentHash []byte
//	TxHash     []byte
//	Height     int64
//	BlockTime  int64
//}
func (chain *BlockChain) ProcGetHeadersMsg(requestblock *types.ReqBlocks) (respheaders *types.Headers, err error) {
	blockhight := chain.GetBlockHeight()

	if requestblock.GetStart() > requestblock.GetEnd() {
		chainlog.Error("ProcGetHeadersMsg input must Start <= End:", "Startheight", requestblock.Start, "Endheight", requestblock.End)
		return nil, types.ErrEndLessThanStartHeight
	}
	if requestblock.End-requestblock.Start >= types.MaxHeaderCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	if requestblock.Start > blockhight {
		chainlog.Error("ProcGetHeadersMsg Startheight err", "startheight", requestblock.Start, "curheight", blockhight)
		return nil, types.ErrStartHeight
	}
	end := requestblock.End
	if requestblock.End > blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetHeadersMsg", "headerscount", count)
	if count < 1 {
		chainlog.Error("ProcGetHeadersMsg count err", "startheight", requestblock.Start, "endheight", requestblock.End, "curheight", blockhight)
		return nil, types.ErrEndLessThanStartHeight
	}

	var headers types.Headers
	headers.Items = make([]*types.Header, count)
	j := 0
	for i := start; i <= end; i++ {
		head, err := chain.blockStore.GetBlockHeaderByHeight(i)
		if err == nil && head != nil {
			headers.Items[j] = head
		} else {
			return nil, err
		}
		j++
	}
	chainlog.Debug("getHeaders", "len", len(headers.Items), "start", start, "end", end)
	return &headers, nil
}

//ProcGetLastHeaderMsg 获取最新区块头信息
func (chain *BlockChain) ProcGetLastHeaderMsg() (*types.Header, error) {
	//首先从缓存中获取最新的blockheader
	head := chain.blockStore.LastHeader()
	if head == nil {
		blockhight := chain.GetBlockHeight()
		tmpHead, err := chain.blockStore.GetBlockHeaderByHeight(blockhight)
		if err == nil && tmpHead != nil {
			chainlog.Error("ProcGetLastHeaderMsg from cache is nil.", "blockhight", blockhight, "hash", common.ToHex(tmpHead.Hash))
			return tmpHead, nil
		}
		return nil, err

	}
	return head, nil
}

/*
ProcGetBlockDetailsMsg EventGetBlocks(types.RequestGetBlock): rpc 模块 会向 blockchain 模块发送 EventGetBlocks(types.RequestGetBlock) 消息，
功能是查询 区块的信息, 回复消息是 EventBlocks(types.Blocks)
type ReqBlocks struct {
	Start int64 `protobuf:"varint,1,opt,name=start" json:"start,omitempty"`
	End   int64 `protobuf:"varint,2,opt,name=end" json:"end,omitempty"`}
type Blocks struct {Items []*Block `protobuf:"bytes,1,rep,name=items" json:"items,omitempty"`}
*/
func (chain *BlockChain) ProcGetBlockDetailsMsg(requestblock *types.ReqBlocks) (respblocks *types.BlockDetails, err error) {
	blockhight := chain.GetBlockHeight()
	if requestblock.Start > blockhight {
		chainlog.Error("ProcGetBlockDetailsMsg Startheight err", "startheight", requestblock.Start, "curheight", blockhight)
		return nil, types.ErrStartHeight
	}
	if requestblock.GetStart() > requestblock.GetEnd() {
		chainlog.Error("ProcGetBlockDetailsMsg input must Start <= End:", "Startheight", requestblock.Start, "Endheight", requestblock.End)
		return nil, types.ErrEndLessThanStartHeight
	}
	if requestblock.End-requestblock.Start >= types.MaxBlockCountPerTime {
		return nil, types.ErrMaxCountPerTime
	}
	chainlog.Debug("ProcGetBlockDetailsMsg", "Start", requestblock.Start, "End", requestblock.End, "Isdetail", requestblock.IsDetail)

	end := requestblock.End
	if requestblock.End > blockhight {
		end = blockhight
	}
	start := requestblock.Start
	count := end - start + 1
	chainlog.Debug("ProcGetBlockDetailsMsg", "blockscount", count)

	var blocks types.BlockDetails
	blocks.Items = make([]*types.BlockDetail, count)
	j := 0
	for i := start; i <= end; i++ {
		block, err := chain.GetBlock(i)
		if err == nil && block != nil {
			if requestblock.IsDetail {
				blocks.Items[j] = block
			} else {
				var blockdetail types.BlockDetail
				blockdetail.Block = block.Block
				blockdetail.Receipts = nil
				blocks.Items[j] = &blockdetail
			}
		} else {
			return nil, err
		}
		j++
	}
	//print
	if requestblock.IsDetail {
		for _, blockinfo := range blocks.Items {
			chainlog.Debug("ProcGetBlocksMsg", "blockinfo", blockinfo.String())
		}
	}
	return &blocks, nil
}

//ProcAddBlockMsg 处理从peer对端同步过来的block消息
func (chain *BlockChain) ProcAddBlockMsg(broadcast bool, blockdetail *types.BlockDetail, pid string) (*types.BlockDetail, error) {
	beg := types.Now()
	defer func() {
		chainlog.Debug("ProcAddBlockMsg", "height", blockdetail.GetBlock().GetHeight(),
			"txCount", blockdetail.GetBlock().GetHeight(), "recvFrom", pid, "cost", types.Since(beg))
	}()

	block := blockdetail.Block
	if block == nil {
		chainlog.Error("ProcAddBlockMsg input block is null")
		return nil, types.ErrInvalidParam
	}
	b, ismain, isorphan, err := chain.ProcessBlock(broadcast, blockdetail, pid, true, -1)
	if b != nil {
		blockdetail = b
	}

	height := blockdetail.Block.GetHeight()
	hash := blockdetail.Block.Hash(chain.client.GetConfig())

	//更新广播block的高度,设置请求过来的区块已经处理完成
	if broadcast {
		chain.UpdateRcvCastBlkHeight(height)
	} else {
		//syncTask 运行时设置对应的blockdone
		if chain.syncTask.InProgress() {
			chain.syncTask.Done(height)
		}
		//downLoadTask 运行时设置对应的blockdone
		if chain.downLoadTask.InProgress() {
			chain.downLoadTask.Done(height)
		}
	}
	if pid == "self" {
		if err != nil {
			return nil, err
		}
		if b == nil {
			return nil, types.ErrExecBlockNil
		}
	}
	chainlog.Debug("ProcAddBlockMsg result:", "height", height, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(hash), "err", err)
	return blockdetail, err
}

//getBlockHashes 获取指定height区间对应的blockhashes
func (chain *BlockChain) getBlockHashes(startheight, endheight int64) types.ReqHashes {
	var reqHashes types.ReqHashes
	for i := startheight; i <= endheight; i++ {
		hash, err := chain.blockStore.GetBlockHashByHeight(i)
		if hash == nil || err != nil {
			storeLog.Error("getBlockHashesByHeight", "height", i, "error", err)
			reqHashes.Hashes = append(reqHashes.Hashes, nil)
		} else {
			reqHashes.Hashes = append(reqHashes.Hashes, hash)
		}
	}
	return reqHashes
}
