// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"strings"

	"github.com/33cn/chain33/types"
)

var (
	filterlog = chainlog.New("submodule", "filter")
)

//GetParaTxByTitle 通过seq以及title获取对应平行连的交易
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
	var reqHashes types.ReqHashes
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

	//通过指定的title过滤对应平行链的交易
	var paraTxs types.ParaTxDetails
	var paraTx *types.ParaTxDetail
	for i, block := range blocks.Items {
		if block != nil {
			paraTx = block.FilterParaTxsByTitle(seq.Title)
			if seq.IsSeq {
				paraTx.Type = sequences.Items[i].GetType()
			} else {
				paraTx.Type = types.AddBlock
			}
		} else {
			paraTx = nil
		}
		paraTxs.Items = append(paraTxs.Items, paraTx)
	}
	return &paraTxs, err
}

//checkInputParam 入参检测，主要检测req的end的值以及title是否合法
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
