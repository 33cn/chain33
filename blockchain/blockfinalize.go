package blockchain

import (
	"encoding/hex"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"sync"
	"time"
)

var (
	blkFinalizeLastChoiceKey = []byte("chain-blockfinalize-lastchoice")
)

type finalizer struct {
	chain  *BlockChain
	choice types.SnowChoice
	lock   sync.RWMutex
	isSync bool
}

func newFinalizer(chain *BlockChain) *finalizer {

	f := &finalizer{chain: chain}

	raw, err := chain.blockStore.db.Get(blkFinalizeLastChoiceKey)

	if err == nil {
		err = types.Decode(raw, &f.choice)
		if err != nil {
			chainlog.Error("newFinalizer", "decode err", err)
			panic(err)
		}
		chainlog.Debug("newFinalizer", "height", f.choice.Height, "hash", hex.EncodeToString(f.choice.Hash))
	} else {
		f.choice.Height = chain.client.GetConfig().GetFork(types.ForkBlockFinalize)
		chainlog.Debug("newFinalizer", "forkHeight", f.choice.Height)
		go f.waitFinalizeStartBlock(f.choice.Height)
	}

	return f
}

func (f *finalizer) waitFinalizeStartBlock(forkHeight int64) {

	for f.chain.blockStore.Height() < forkHeight+12 {
		time.Sleep(time.Second * 5)
	}

	detail, err := f.chain.GetBlock(forkHeight)
	if err != nil {
		chainlog.Error("setFinalizedStartHeight", "height", forkHeight, "get block err", err)
		panic(err)
	}
	_ = f.setFinalizedBlock(detail.GetBlock().Height, detail.GetBlock().Hash(f.chain.client.GetConfig()))
}

func (f *finalizer) snowmanPreferBlock(msg *queue.Message) {
	req := (msg.Data).(*types.ReqBytes)

	detail, err := f.chain.LoadBlockByHash(req.GetData())
	if err != nil {
		chainlog.Warn("snowmanPreferBlock", "hash", hex.EncodeToString(req.GetData()), "load block err", err.Error())
		return
	}
	chainlog.Debug("snowmanPreferBlock", "height", detail.GetBlock().GetHeight(), "hash", hex.EncodeToString(req.GetData()))

	return

}

func (f *finalizer) snowmanAcceptBlock(msg *queue.Message) {

	req := (msg.Data).(*types.SnowChoice)
	chainlog.Debug("snowmanAcceptBlock", "height", req.Height, "hash", hex.EncodeToString(req.Hash))
	height, _ := f.getLastFinalized()
	if req.GetHeight() <= height {
		chainlog.Debug("snowmanAcceptBlock disorder", "height", req.Height, "hash", hex.EncodeToString(req.Hash))
		return
	}
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

	chainlog.Debug("setFinalizedBlock", "height", height, "hash", hex.EncodeToString(hash))
	f.lock.Lock()
	defer f.lock.Unlock()
	f.choice.Height = height
	f.choice.Hash = hash
	err := f.chain.blockStore.db.Set(blkFinalizeLastChoiceKey, types.Encode(&f.choice))
	if err != nil {
		chainlog.Error("setFinalizedBlock", "height", height, "hash", hex.EncodeToString(hash), "err", err)
		return err
	}
	return nil
}

func (f *finalizer) getLastFinalized() (int64, []byte) {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.choice.Height, f.choice.Hash
}

func (f *finalizer) snowmanLastChoice(msg *queue.Message) {

	height, hash := f.getLastFinalized()
	chainlog.Debug("snowmanLastChoice", "height", height, "hash", hex.EncodeToString(hash))
	msg.Reply(f.chain.client.NewMessage(msg.Topic,
		types.EventSnowmanLastChoice, &types.SnowChoice{Height: height, Hash: hash}))
}

func (f *finalizer) setSyncStatus(status bool) {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.isSync = status
}

func (f *finalizer) getSyncStatus() bool {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return f.isSync
}

const finlizeChoiceMaxInterval = 128

func (f *finalizer) finalizedStateSync() {

	//for !f.chain.IsCaughtUp() {
	//	chainlog.Debug("finalizedStateSync wait chain sync")
	//	time.Sleep(time.Second * 3)
	//}
	//
	//minPeerCount := snowball.DefaultParameters.K
	//
	//for count := f.chain.GetPeerCount(); count < minPeerCount; {
	//	chainlog.Debug("finalizedStateSync wait more peers", "count", count)
	//	time.Sleep(time.Second * 3)
	//}
	//currHeight := f.chain.blockStore.Height()
	//finalized, _ := f.getLastFinalized()
	//if finalized >= currHeight-finlizeChoiceMaxInterval {
	//	f.setSyncStatus(true)
	//	return
	//}
	//
	//detail, err := f.chain.GetBlock(currHeight - finlizeChoiceMaxInterval/2)
	//if err != nil {
	//	chainlog.Error("finalizedStateSync", "GetBlock err:", err)
	//	return
	//}
	//// query snow choice
	//req := &types.SnowChoice{
	//	Height: detail.Block.Height,
	//	Hash:   detail.Block.Hash(f.chain.client.GetConfig()),
	//}
	//
	//msg := f.chain.client.NewMessage("p2p", types.EventSnowmanQueryChoice, req)
	//err = f.chain.client.Send(msg, true)
	//if err != nil {
	//	chainlog.Error("finalizedStateSync", "client.Send err:", err)
	//	return
	//}
	//
	//replyMsg, err := f.chain.client.Wait(msg)
	//if err != nil {
	//	chainlog.Error("finalizedStateSync", "client.Wait err:", err)
	//	return
	//}

}
