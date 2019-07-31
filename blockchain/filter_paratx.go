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

	//获取区块的seq信息
	req := &types.ReqBlocks{Start: seq.Start, End: seq.End, IsDetail: false, Pid: []string{}}
	sequences, err := chain.GetBlockSequences(req)
	if err != nil {
		filterlog.Error("GetParaTxByTitle:GetBlockSequences", "err", err.Error())
		return nil, err
	}

	//通过区块hash获取区块信息
	var reqHashes types.ReqHashes
	for _, item := range sequences.Items {
		if item != nil {
			reqHashes.Hashes = append(reqHashes.Hashes, item.GetHash())
		}
	}

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
			paraTx.Type = sequences.Items[i].GetType()
		} else {
			paraTx = nil
		}
		paraTxs.Items = append(paraTxs.Items, paraTx)
	}
	return &paraTxs, err
}

//checkInputParam 入参检测，主要检测seq的end的值已经title是否合法
func (chain *BlockChain) checkInputParam(seq *types.ReqParaTxByTitle) error {

	//入参数校验
	blockLastSeq, err := chain.blockStore.LoadBlockLastSequence()
	if err != nil || seq.End > blockLastSeq || blockLastSeq < 0 || !strings.HasPrefix(seq.Title, types.ParaKeyX) {
		filterlog.Error("checkInputParam", "blockLastSeq", blockLastSeq, "seq", seq, "err", err)
		return types.ErrInvalidParam
	}
	return nil
}
