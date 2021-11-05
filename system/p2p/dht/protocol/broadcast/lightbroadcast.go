// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"container/list"
	"sync"

	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

type ltBroadcast struct {
	*broadcastProtocol
	ltBlockList      *list.List
	blockRequestList *list.List
	ltBlockLock      sync.RWMutex
	blockReqLock     sync.RWMutex
}

// 等待组装的轻区块信息
type lightBlock struct {
	fromPeer          peer.ID
	receiveTimeStamp  int64
	totalBlock        *types.Block
	ltBlock           *types.LightBlock
	notExistTxHashes  []string
	notExistTxIndices []int
}

// 获取区块请求
type blockRequest struct {
	fromPeer  peer.ID
	blockHash []byte
}

func newLtBroadcast(b *broadcastProtocol) *ltBroadcast {
	l := &ltBroadcast{broadcastProtocol: b}
	l.ltBlockList = list.New()
	l.blockRequestList = list.New()
	return l
}

func (l *ltBroadcast) broadcast() {

	go l.handleLtBlockList()
	go l.handleBlockRequestList()
}

const (
	defaultLtBlockTimeout = 2 //seconds
)

func (l *ltBroadcast) buildTotalBlock(ltBlock *lightBlock) bool {

	if len(ltBlock.notExistTxHashes) == 0 {
		return true
	}

	//get tx list from mempool
	resp, err := l.P2PEnv.QueryModule("mempool", types.EventTxListByHash,
		&types.ReqTxHashList{Hashes: ltBlock.notExistTxHashes, IsShortHash: true})
	if err != nil {
		log.Error("buildTotalBlock", "queryTxListByHashErr", err)
		return false
	}

	txList, ok := resp.(*types.ReplyTxList)
	if !ok {
		log.Error("buildTotalBlock", "queryMemPool", "nilReplyTxList")
		return false
	}

	index := 0
	for _, tx := range ltBlock.totalBlock.GetTxs() {
		if index > len(txList.GetTxs()) {
			break
		}

		if tx == nil {
			tx = txList.GetTxs()[index]
			index++
		}
	}
	return true
}

func (l *ltBroadcast) addLtBlock(ltBlock *types.LightBlock, receiveFrom peer.ID) {

	//组装block
	block := &types.Block{}
	block.SetHeader(ltBlock.Header)
	block.Txs = make([]*types.Transaction, 0, ltBlock.Header.TxCount)
	//add miner tx
	block.Txs = append(block.Txs, ltBlock.MinerTx)

	lb := &lightBlock{
		fromPeer:         receiveFrom,
		totalBlock:       block,
		ltBlock:          ltBlock,
		receiveTimeStamp: types.Now().Unix(),
		notExistTxHashes: ltBlock.STxHashes,
	}

	l.ltBlockLock.Lock()
	defer l.ltBlockLock.Unlock()

	l.ltBlockList.PushBack(lb)
}

func (l *ltBroadcast) addBlockRequest(req *types.ReqBytes, receiveFrom peer.ID) {

}

func (l *ltBroadcast) handleLtBlockList() {

	for {
		select {
		case <-l.Ctx.Done():
			return
		}
	}
}

func (l *ltBroadcast) handleBlockRequestList() {

}
