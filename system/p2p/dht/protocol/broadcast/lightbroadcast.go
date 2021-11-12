// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package broadcast

import (
	"bytes"
	"container/list"
	"encoding/hex"
	"sync"
	"time"

	"github.com/33cn/chain33/common"

	"github.com/33cn/chain33/types"
	"github.com/libp2p/go-libp2p-core/peer"
)

type ltBroadcast struct {
	*broadcastProtocol
	pendBlockList    *list.List
	blockRequestList *list.List
	pdBlockLock      sync.RWMutex
	blockReqLock     sync.RWMutex
}

// 等待组装的轻区块信息
type pendBlock struct {
	fromPeer          peer.ID
	receiveTimeStamp  int64
	block             *types.Block
	blockDataHash     []byte
	blockHash         []byte
	sTxHashes         []string
	notExistTxHashes  []string
	notExistTxIndices []int
}

// 获取区块请求
type blockRequest struct {
	fromPeer    peer.ID
	blockHeight int64
}

func newLtBroadcast(b *broadcastProtocol) *ltBroadcast {
	l := &ltBroadcast{broadcastProtocol: b}
	l.pendBlockList = list.New()
	l.blockRequestList = list.New()
	return l
}

func (l *ltBroadcast) broadcast() {

	go l.pendBlockLoop()
	go l.blockRequestLoop()
}

func (l *ltBroadcast) buildPendBlock(pd *pendBlock) bool {

	if len(pd.sTxHashes) == 0 {
		return true
	}
	pd.notExistTxHashes = pd.notExistTxHashes[:0]
	pd.notExistTxIndices = pd.notExistTxIndices[:0]
	for i, tx := range pd.block.GetTxs()[1:] {
		if tx == nil {
			pd.notExistTxIndices = append(pd.notExistTxIndices, i+1)
			pd.notExistTxHashes = append(pd.notExistTxHashes, pd.sTxHashes[i])
		}
	}

	//get tx list from mempool
	resp, err := l.P2PEnv.QueryModule("mempool", types.EventTxListByHash,
		&types.ReqTxHashList{Hashes: pd.notExistTxHashes, IsShortHash: true})
	if err != nil {
		log.Error("buildPendBlock", "queryTxListByHashErr", err)
		return false
	}

	txList, ok := resp.(*types.ReplyTxList)
	if !ok {
		log.Error("buildPendBlock", "queryMemPool", "nilReplyTxList")
		return false
	}
	// 请求mempool会返回相应长度的数组
	for i := 0; i < len(pd.notExistTxIndices); i++ {
		tx := txList.GetTxs()[i]
		if tx == nil {
			continue
		}
		index := pd.notExistTxIndices[i]
		pd.block.GetTxs()[index] = tx
		// 交易组处理
		group, _ := tx.GetTxGroup()
		// 交易组中的其他交易, 依次添加到区块交易列表中
		for j, tx := range group.GetTxs() {
			pd.block.GetTxs()[index+j] = tx
		}
	}

	if bytes.Equal(pd.blockDataHash, common.Sha256(types.Encode(pd.block))) {
		blockHashHex := hex.EncodeToString(pd.blockHash)
		log.Debug("buildLtBlock", "height", pd.block.GetHeight(), "hash", blockHashHex,
			"wait(s)", types.Now().Unix()-pd.receiveTimeStamp)
		_ = l.postBlockChain(blockHashHex, pd.fromPeer.String(), pd.block)
		return true
	}
	return false
}

func (l *ltBroadcast) addLtBlock(ltBlock *types.LightBlock, receiveFrom peer.ID) {

	//组装block
	block := &types.Block{}
	block.SetHeader(ltBlock.GetHeader())
	txCount := ltBlock.GetHeader().GetTxCount()
	block.Txs = make([]*types.Transaction, txCount)
	//add miner tx
	block.Txs[0] = ltBlock.MinerTx

	pd := &pendBlock{
		fromPeer:          receiveFrom,
		block:             block,
		sTxHashes:         ltBlock.GetSTxHashes(),
		receiveTimeStamp:  types.Now().Unix(),
		blockHash:         ltBlock.GetHeader().GetHash(),
		blockDataHash:     ltBlock.GetBlockDataHash(),
		notExistTxIndices: make([]int, txCount),
		notExistTxHashes:  make([]string, txCount),
	}

	if l.buildPendBlock(pd) {
		return
	}

	l.pdBlockLock.Lock()
	defer l.pdBlockLock.Unlock()

	l.pendBlockList.PushBack(pd)
}

const (
	defaultLtBlockTimeout = 2 //seconds
)

func (l *ltBroadcast) buildPendList() []*pendBlock {

	l.pdBlockLock.Lock()
	defer l.pdBlockLock.Unlock()

	removeItems := make([]*list.Element, 0)
	timeoutBlocks := make([]*pendBlock, 0)
	for it := l.pendBlockList.Front(); it != nil; it = it.Next() {
		pd := it.Value.(*pendBlock)
		pendTime := types.Now().Unix() - pd.receiveTimeStamp
		if l.buildPendBlock(pd) {
			log.Debug("buildPendSuccess", "height", pd.block.GetHeight(), "wait", pendTime)
			removeItems = append(removeItems, it)
		} else if pendTime >= l.cfg.LtBlockPendTimeout {
			log.Debug("buildPendTimeout", "height", pd.block.GetHeight())
			removeItems = append(removeItems, it)
			timeoutBlocks = append(timeoutBlocks, pd)
		}
	}
	for _, item := range removeItems {
		l.pendBlockList.Remove(item)
	}
	return timeoutBlocks
}

func (l *ltBroadcast) pendBlockLoop() {

	ticker := time.NewTicker(time.Millisecond * 200)
	for {
		select {
		case <-l.Ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			pdBlocks := l.buildPendList()
			for _, pd := range pdBlocks {
				l.pubPeerMsg(pd.fromPeer, blockReqMsgID, &types.ReqInt{Height: pd.block.GetHeight()})
			}
		}
	}
}

func (l *ltBroadcast) handleBlockReq(req *blockRequest) bool {
	// 当前高度小于请求高度, 继续等待
	if l.getCurrentHeight() < req.blockHeight {
		return false
	}
	// 向blockchain模块请求区块
	details, err := l.API.GetBlocks(&types.ReqBlocks{Start: req.blockHeight, End: req.blockHeight})
	if err != nil {
		log.Error("handleBlockReq", "height", req.blockHeight, "get block err", err)
	} else {
		l.pubPeerMsg(req.fromPeer, blockRespMsgID, details.GetItems()[0].Block)
	}

	return true
}

func (l *ltBroadcast) addBlockRequest(height int64, receiveFrom peer.ID) {

	if height <= 0 {
		return
	}
	req := &blockRequest{
		blockHeight: height,
		fromPeer:    receiveFrom,
	}

	if l.handleBlockReq(req) {
		return
	}

	l.blockReqLock.Lock()
	defer l.blockReqLock.Unlock()
	l.blockRequestList.PushBack(req)
}

func (l *ltBroadcast) handleBlockReqList() {
	l.blockReqLock.Lock()
	defer l.blockReqLock.Unlock()

	removeItems := make([]*list.Element, 0)
	for it := l.blockRequestList.Front(); it != nil; it = it.Next() {

		req := it.Value.(*blockRequest)
		if l.handleBlockReq(req) {
			removeItems = append(removeItems, it)
		}
	}

	for _, item := range removeItems {
		l.blockRequestList.Remove(item)
	}

}

func (l *ltBroadcast) blockRequestLoop() {

	ticker := time.NewTicker(time.Millisecond * 200)
	for {
		select {
		case <-l.Ctx.Done():
			ticker.Stop()
			return
		case <-ticker.C:
			l.handleBlockReqList()
		}
	}
}
