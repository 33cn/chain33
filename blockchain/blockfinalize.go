package blockchain

import (
	"encoding/hex"
	"sync"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	blkFinalizingStartHeight    int64
	blkFinalizeLastAcceptBlkKey = []byte("chain-blockfinalize-acceptblk")
)

type finalizer struct {
	chain  *BlockChain
	header types.Header
	lock   sync.RWMutex
}

func newFinalizer(chain *BlockChain) *finalizer {

	f := &finalizer{chain: chain}

	raw, err := chain.blockStore.db.Get(blkFinalizeLastAcceptBlkKey)

	if err == nil {
		types.Decode(raw, &f.header)
		return f
	}

	detail, err := chain.GetBlock(blkFinalizingStartHeight)
	if err != nil {
		chainlog.Error("newFinalizer", "height", blkFinalizingStartHeight, "get block err", err)
		panic(err)
	}

	f.header.Hash = detail.GetBlock().Hash(chain.client.GetConfig())
	f.header.Height = detail.GetBlock().GetHeight()
	return f
}

func (f *finalizer) syncNeighborsFinalizedHeader(selfHeight int64) {

}

func (f *finalizer) eventPreferBlock(msg *queue.Message) {
	req := (msg.Data).(*types.ReqBytes)

	detail, err := f.chain.LoadBlockByHash(req.GetData())
	if err != nil {
		chainlog.Error("eventPrferBlock", "hash", hex.EncodeToString(req.GetData()), "load block err", err.Error())
		return
	}
	chainlog.Debug("eventPreferBlock", "height", detail.GetBlock().GetHeight(), "hash", hex.EncodeToString(req.GetData()))

	return

}

func (f *finalizer) eventAcceptBlock(msg *queue.Message) {

	req := (msg.Data).(*types.ReqBytes)
	detail, err := f.chain.LoadBlockByHash(req.GetData())
	if err != nil {
		chainlog.Error("eventAcceptBlock", "hash", hex.EncodeToString(req.GetData()), "load block err", err.Error())
		return
	}

	chainlog.Debug("eventAcceptBlock", "height", detail.GetBlock().GetHeight(), "hash", hex.EncodeToString(req.GetData()))
	err = f.setFinalizedBlock(detail.GetBlock().GetHeight(), req.GetData())

	if err != nil {
		chainlog.Error("eventAcceptBlock", "setFinalizedBlock err", err.Error())
	}
}

func (f *finalizer) setFinalizedBlock(height int64, hash []byte) error {

	f.lock.Lock()
	defer f.lock.Unlock()
	err := f.chain.blockStore.db.Set(blkFinalizeLastAcceptBlkKey, types.Encode(&f.header))

	if err != nil {
		return err
	}
	f.header.Height = height
	f.header.Hash = hash
	return nil
}

func (f *finalizer) getFinalizedHeight() int64 {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.header.Height
}

func (f *finalizer) eventLastAcceptHeight(msg *queue.Message) {

	height := f.getFinalizedHeight()
	chainlog.Debug("eventLastAcceptHeight", "height", height)
	msg.Reply(f.chain.client.NewMessage("consensus", types.EventSnowmanLastAcceptHeight, &types.Int64{Data: height}))
}
