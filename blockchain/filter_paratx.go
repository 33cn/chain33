// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"strings"

	"github.com/33cn/chain33/common/merkle"
	"github.com/33cn/chain33/types"
)

var (
	filterlog = chainlog.New("submodule", "filter")
)

// GetParaTxByTitle 通过seq以及title获取对应平行连的交易
func (chain *BlockChain) GetParaTxByTitle(seq *types.ReqParaTxByTitle) (*types.ParaTxDetails, error) {

	//入参数校验
	err := chain.checkInputParam(seq)
	if err != nil {
		return nil, err
	}

	//对获取区块的起始和结束值做校验,最多一次取1000个区块，防止取的数据过大导致内存异常
	if seq.End > seq.Start && (seq.End-seq.Start > types.MaxBlockCountPerTime) {
		seq.End = seq.Start + types.MaxBlockCountPerTime - 1
	}
	reqHashes := &types.ReqHashes{}
	var sequences *types.BlockSequences
	if seq.IsSeq {
		req := &types.ReqBlocks{Start: seq.Start, End: seq.End, IsDetail: false, Pid: []string{}}
		sequences, err = chain.GetBlockSequences(req)
		if err != nil {
			filterlog.Error("GetParaTxByTitle:GetBlockSequences", "err", err.Error())
			return nil, err
		}
		for _, item := range sequences.Items {
			if item != nil {
				reqHashes.Hashes = append(reqHashes.Hashes, item.GetHash())
			}
		}
	} else {
		reqHashes = chain.getBlockHashes(seq.Start, seq.End)
	}

	//通过区块hash获取区块信息
	blocks, err := chain.GetBlockByHashes(reqHashes.Hashes)
	if err != nil {
		filterlog.Error("GetParaTxByTitle:GetBlockByHashes", "err", err)
		return nil, err
	}

	cfg := chain.client.GetConfig()
	//通过指定的title过滤对应平行链的交易
	var paraTxs types.ParaTxDetails
	var paraTx *types.ParaTxDetail
	for i, block := range blocks.Items {
		if block != nil {
			paraTx = block.FilterParaTxsByTitle(cfg, seq.Title)
			if seq.IsSeq {
				paraTx.Type = sequences.Items[i].GetType()
			} else {
				paraTx.Type = types.AddBlock
			}
			height := block.Block.GetHeight()
			blockhash := block.Block.Hash(cfg)
			if cfg.IsFork(height, "ForkRootHash") {
				branch, childHash, index := chain.getChildChainProofs(height, blockhash, seq.Title, block.Block.GetTxs())
				paraTx.ChildHash = childHash
				paraTx.Index = index
				paraTx.Proofs = branch
			}
		} else {
			paraTx = nil
		}
		paraTxs.Items = append(paraTxs.Items, paraTx)
	}
	return &paraTxs, nil
}

// checkInputParam 入参检测，主要检测req的end的值以及title是否合法
func (chain *BlockChain) checkInputParam(req *types.ReqParaTxByTitle) error {
	var err error
	var lastblock int64

	if req.GetStart() > req.GetEnd() {
		chainlog.Error("checkInputParam input must Start <= End:", "Start", req.Start, "End", req.End)
		return types.ErrEndLessThanStartHeight
	}

	//需要区分是通过seq/height来获取平行链交易
	if req.IsSeq {
		lastblock, err = chain.blockStore.LoadBlockLastSequence()
	} else {
		lastblock = chain.GetBlockHeight()
	}
	if err != nil || req.End > lastblock || lastblock < 0 || !strings.HasPrefix(req.Title, types.ParaKeyX) {
		filterlog.Error("checkInputParam", "lastblock", lastblock, "req", req, "err", err)
		return types.ErrInvalidParam
	}
	return nil
}

// GetParaTxByHeight 通过height以及title获取对应平行连的交易
func (chain *BlockChain) GetParaTxByHeight(req *types.ReqParaTxByHeight) (*types.ParaTxDetails, error) {
	//入参数校验
	if req == nil {
		return nil, types.ErrInvalidParam
	}
	count := len(req.Items)
	if int64(count) > types.MaxBlockCountPerTime {
		return nil, types.ErrInvalidParam
	}
	lastheight := chain.GetBlockHeight()
	if req.Items[0] < 0 || req.Items[count-1] > lastheight {
		return nil, types.ErrInvalidParam
	}
	if !strings.HasPrefix(req.Title, types.ParaKeyX) {
		return nil, types.ErrInvalidParam
	}
	cfg := chain.client.GetConfig()
	var paraTxs types.ParaTxDetails
	var paraTxDetail *types.ParaTxDetail
	size := 0
	for _, height := range req.Items {
		block, err := chain.GetBlock(height)
		if err != nil {
			filterlog.Error("GetParaTxByHeight:GetBlock", "height", height, "err", err)
		} else {
			paraTxDetail = block.FilterParaTxsByTitle(cfg, req.Title)
			if paraTxDetail != nil {
				paraTxDetail.Type = types.AddBlock
				blockHash := block.Block.Hash(cfg)
				if cfg.IsFork(height, "ForkRootHash") {
					branch, childHash, index := chain.getChildChainProofs(height, blockHash, req.Title, block.Block.GetTxs())
					paraTxDetail.ChildHash = childHash
					paraTxDetail.Index = index
					paraTxDetail.Proofs = branch
				}
			}
		}
		size += paraTxDetail.Size()
		if size > types.MaxBlockSizePerTime {
			chainlog.Info("GetParaTxByHeight:overflow", "MaxBlockSizePerTime", types.MaxBlockSizePerTime)
			return &paraTxs, nil
		}
		paraTxs.Items = append(paraTxs.Items, paraTxDetail)
	}
	return &paraTxs, nil
}

// 获取指定title子链roothash在指定高度上的路径证明
func (chain *BlockChain) getChildChainProofs(height int64, blockHash []byte, title string, txs []*types.Transaction) ([][]byte, []byte, uint32) {
	var branch [][]byte
	var childHash []byte
	var index uint32
	var hashes [][]byte
	var exist bool
	//主链直接从数据库获取对应的子链根hash
	paraTxs, err := chain.LoadParaTxByHeight(height, "", 0, 1)
	if err == nil && len(paraTxs.Items) > 0 && bytes.Equal(paraTxs.Items[0].GetHash(), blockHash) {
		for _, paratx := range paraTxs.Items {
			if title == paratx.Title {
				index = paratx.ChildHashIndex
				childHash = paratx.ChildHash
				exist = true
			}
			hashes = append(hashes, paratx.ChildHash)
		}
		if exist {
			branch = merkle.GetMerkleBranch(hashes, index)
			return branch, childHash, index
		}
		return nil, nil, 0
	}
	//侧链的需要重新计算
	cfg := chain.client.GetConfig()
	_, childChains := merkle.CalcMultiLayerMerkleInfo(cfg, height, txs)
	for i, childchain := range childChains {
		hashes = append(hashes, childchain.ChildHash)
		if childchain.Title == title {
			index = uint32(i)
			childHash = childchain.ChildHash
			exist = true
		}
	}
	if exist {
		branch = merkle.GetMerkleBranch(hashes, index)
		return branch, childHash, index
	}
	return nil, nil, 0
}

// LoadParaTxByTitle 通过title获取本平行链交易所在的区块高度信息前后翻页
func (chain *BlockChain) LoadParaTxByTitle(req *types.ReqHeightByTitle) (*types.ReplyHeightByTitle, error) {
	var primaryKey []byte

	//入参合法性检测
	if req == nil || int64(req.Count) > types.MaxHeaderCountPerTime {
		return nil, types.ErrInvalidParam
	}
	if req.GetDirection() != 0 && req.GetDirection() != 1 {
		return nil, types.ErrInvalidParam
	}
	curHeight := chain.GetBlockHeight()
	if req.Height > curHeight || len(req.Title) == 0 || !strings.HasPrefix(req.Title, types.ParaKeyX) {
		return nil, types.ErrInvalidParam
	}
	if req.Height != -1 {
		primaryKey = calcHeightTitleKey(req.Height, req.Title)
	} else {
		primaryKey = nil
	}
	paraInfos, err := getParaTxByIndex(chain.blockStore.db, "title", []byte(req.Title), primaryKey, req.Count, req.Direction)
	if err != nil {
		return nil, err
	}

	var reply types.ReplyHeightByTitle
	reply.Title = req.Title
	for _, para := range paraInfos.Items {
		reply.Items = append(reply.Items, &types.BlockInfo{Height: para.GetHeight(), Hash: para.GetHash()})
	}
	return &reply, nil
}

// LoadParaTxByHeight 通过height获取本高度上的平行链title前后翻页;预留接口
func (chain *BlockChain) LoadParaTxByHeight(height int64, title string, count, direction int32) (*types.HeightParas, error) {
	//入参合法性检测
	curHeight := chain.GetBlockHeight()
	if height > curHeight || height < 0 {
		return nil, types.ErrInvalidParam
	}

	var primaryKey []byte
	if len(title) == 0 {
		primaryKey = nil
	} else if !strings.HasPrefix(title, types.ParaKeyX) {
		return nil, types.ErrInvalidParam
	} else {
		primaryKey = calcHeightTitleKey(height, title)
	}
	return getParaTxByIndex(chain.blockStore.db, "height", calcHeightParaKey(height), primaryKey, count, direction)
}
