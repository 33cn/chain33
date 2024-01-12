package blockchain

import (
	"encoding/hex"
	"sync"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	blockFinalizeStartHeight    int64
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
	}

	if chain.blockStore.Height() >= blockFinalizeStartHeight {

		detail, err := chain.GetBlock(blockFinalizeStartHeight)
		if err != nil {
			chainlog.Error("newFinalizer", "height", blockFinalizeStartHeight, "get block err", err)
			panic(err)
		}
		f.setFinalizedBlock(detail.GetBlock().Height, detail.GetBlock().Hash(chain.client.GetConfig()))
	}

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

func (f *finalizer) getFinalizedBlock() (int64, []byte) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.header.Height, f.header.Hash
}

func (f *finalizer) eventLastAcceptHeight(msg *queue.Message) {

	height, hash := f.getFinalizedBlock()
	chainlog.Debug("eventLastAcceptHeight", "height", height, "hash", hex.EncodeToString(hash))
	msg.Reply(f.chain.client.NewMessage("consensus", types.EventSnowmanLastAcceptHeight, &types.ReqBytes{Data: hash}))
}
