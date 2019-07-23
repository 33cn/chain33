// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
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
		chainlog.Error("GetParaTxByTitle:GetBlockSequences", "err", err.Error())
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
		chainlog.Error("GetParaTxByTitle:GetBlockByHashes", "err", err)
		return nil, err
	}

	//通过指定的title过滤对应平行链的交易
	var paraTxs types.ParaTxDetails
	var paraTx *types.ParaTxDetail
	for i, block := range blocks.Items {
		if block != nil {
			paraTx = filterParaTxsByTitle(seq.Title, block)
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
		chainlog.Error("checkInputParam", "blockLastSeq", blockLastSeq, "seq", seq, "err", err)
		return types.ErrInvalidParam
	}
	return nil
}

//filterParaTxsByTitle 过滤指定title的平行链交易
//1，单笔平行连交易
//2,交易组中的平行连交易，需要将整个交易组都过滤出来
//目前暂时不返回单个交易的proof证明路径，
//后面会将平行链的交易组装到一起，构成一个子roothash。会返回子roothash的proof证明路径
func filterParaTxsByTitle(title string, blockDetail *types.BlockDetail) *types.ParaTxDetail {
	var paraTx types.ParaTxDetail
	paraTx.Header = blockDetail.Block.GetHeader()

	for i := 0; i < len(blockDetail.Block.Txs); i++ {
		tx := blockDetail.Block.Txs[i]
		if types.IsSpecificParaExecName(title, string(tx.Execer)) {

			//过滤交易组中的para交易，需要将整个交易组都过滤出来
			if tx.GroupCount >= 2 {
				txDetails, endIdx := filterParaTxGroup(tx, blockDetail, i)
				paraTx.TxDetails = append(paraTx.TxDetails, txDetails...)
				i = endIdx - 1
				continue
			}

			//单笔para交易
			var txDetail types.TxDetail
			txDetail.Tx = tx
			txDetail.Receipt = blockDetail.Receipts[i]
			txDetail.Index = uint32(i)
			paraTx.TxDetails = append(paraTx.TxDetails, &txDetail)

		}
	}
	return &paraTx
}

//filterParaTxGroup 获取para交易所在交易组信息
func filterParaTxGroup(tx *types.Transaction, blockDetail *types.BlockDetail, index int) ([]*types.TxDetail, int) {
	var headIdx int
	var txDetails []*types.TxDetail

	for i := index; i >= 0; i-- {
		if bytes.Equal(tx.Header, blockDetail.Block.Txs[i].Hash()) {
			headIdx = i
			break
		}
	}

	endIdx := headIdx + int(tx.GroupCount)
	for i := headIdx; i < endIdx; i++ {
		var txDetail types.TxDetail
		txDetail.Tx = blockDetail.Block.Txs[i]
		txDetail.Receipt = blockDetail.Receipts[i]
		txDetail.Index = uint32(i)
		txDetails = append(txDetails, &txDetail)
	}
	return txDetails, endIdx
}
