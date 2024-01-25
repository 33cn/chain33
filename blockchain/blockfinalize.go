package blockchain

import (
	"encoding/hex"
	"sync"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	blockFinalizeStartHeight int64
	blkFinalizeLastChoiceKey = []byte("chain-blockfinalize-lastchoice")
)

type finalizer struct {
	chain  *BlockChain
	header types.Header
	lock   sync.RWMutex
}

func newFinalizer(chain *BlockChain) *finalizer {

	f := &finalizer{chain: chain}

	raw, err := chain.blockStore.db.Get(blkFinalizeLastChoiceKey)

	if err == nil {
		types.Decode(raw, &f.header)
	} else if chain.blockStore.Height() >= blockFinalizeStartHeight {

		detail, err := chain.GetBlock(blockFinalizeStartHeight)
		if err != nil {
			chainlog.Error("newFinalizer", "height", blockFinalizeStartHeight, "get block err", err)
			panic(err)
		}
		_ = f.setFinalizedBlock(detail.GetBlock().Height, detail.GetBlock().Hash(chain.client.GetConfig()))
	}

	chainlog.Debug("newFinalizer", "height", f.header.Height, "hash", hex.EncodeToString(f.header.Hash))
	return f
}

func (f *finalizer) syncNeighborsFinalizedHeader(selfHeight int64) {

}

func (f *finalizer) snowmanPreferBlock(msg *queue.Message) {
	req := (msg.Data).(*types.ReqBytes)

	detail, err := f.chain.LoadBlockByHash(req.GetData())
	if err != nil {
		chainlog.Error("snowmanPreferBlock", "hash", hex.EncodeToString(req.GetData()), "load block err", err.Error())
		return
	}
	chainlog.Debug("snowmanPreferBlock", "height", detail.GetBlock().GetHeight(), "hash", hex.EncodeToString(req.GetData()))

	return

}

func (f *finalizer) snowmanAcceptBlock(msg *queue.Message) {

	req := (msg.Data).(*types.SnowChoice)

	chainlog.Debug("snowmanAcceptBlock", "height", req.Height, "hash", hex.EncodeToString(req.Hash))
	detail, err := f.chain.LoadBlockByHash(req.GetHash())
	if err != nil {
		chainlog.Error("snowmanAcceptBlock", "height", req.Height,
			"hash", hex.EncodeToString(req.GetHash()), "load block err", err.Error())
		return
	}

	if detail.GetBlock().GetHeight() != req.GetHeight() {

		chainlog.Error("snowmanAcceptBlock height not equal", "expect", req.Height, "actual", detail.GetBlock().GetHeight(),
			"hash", hex.EncodeToString(req.GetHash()))
		return
	}

	err = f.setFinalizedBlock(detail.GetBlock().GetHeight(), req.GetHash())

	if err != nil {
		chainlog.Error("snowmanAcceptBlock", "setFinalizedBlock err", err.Error())
	}
}

func (f *finalizer) setFinalizedBlock(height int64, hash []byte) error {

	f.lock.Lock()
	defer f.lock.Unlock()
	err := f.chain.blockStore.db.Set(blkFinalizeLastChoiceKey, types.Encode(&f.header))

	if err != nil {
		chainlog.Error("setFinalizedBlock", "height", height, "hash", hex.EncodeToString(hash), "err", err)
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

func (f *finalizer) snowmanLastChoice(msg *queue.Message) {

	height, hash := f.getFinalizedBlock()
	chainlog.Debug("snowmanLastChoice", "height", height, "hash", hex.EncodeToString(hash))
	msg.Reply(f.chain.client.NewMessage(msg.Topic,
		types.EventSnowmanLastChoice, &types.SnowChoice{Height: height, Hash: hash}))
}
