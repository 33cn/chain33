package blockchain

import (
	"encoding/hex"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"sync"
	"time"
)

var (
	snowChoiceKey = []byte("blockchain-snowchoice")
)

type finalizer struct {
	chain  *BlockChain
	choice types.SnowChoice
	lock   sync.RWMutex
}

func (f *finalizer) Init(chain *BlockChain) {

	f.chain = chain
	raw, err := chain.blockStore.db.Get(snowChoiceKey)

	if err == nil {
		err = types.Decode(raw, &f.choice)
		if err != nil {
			chainlog.Error("newFinalizer", "decode err", err)
			panic(err)
		}
		chainlog.Info("newFinalizer", "height", f.choice.Height, "hash", hex.EncodeToString(f.choice.Hash))
		go f.healthCheck(f.choice.Height)
	} else if chain.client.GetConfig().GetModuleConfig().Consensus.Finalizer != "" {
		f.choice.Height = chain.cfg.BlockFinalizeEnableHeight
		chainlog.Info("newFinalizer", "enableHeight", f.choice.Height, "gapHeight", chain.cfg.BlockFinalizeGapHeight)
		go f.waitFinalizeStartBlock(f.choice.Height)
	}
}

func (f *finalizer) healthCheck(finalizedHeight int64) {

	ticker := time.NewTicker(time.Minute * 3)
	for {

		select {

		case <-f.chain.quit:
			return

		case <-ticker.C:

			chainHeight := f.chain.bestChain.Height()
			height, _ := f.getLastFinalized()
			chainlog.Debug("finalizer healthCheck", "lastFinalize", finalizedHeight, "currFinalize", height, "chainHeight", chainHeight)
			if height > finalizedHeight || chainHeight <= height {
				finalizedHeight = height
				continue
			}
			detail, err := f.chain.GetBlock(height)
			if err != nil {
				chainlog.Error("finalizer healthCheck", "height", height, "get block err", err)
				continue
			}
			_ = f.setFinalizedBlock(detail.GetBlock().Height, detail.GetBlock().Hash(f.chain.client.GetConfig()), false)
			_ = f.chain.client.Send(queue.NewMessage(types.EventSnowmanResetEngine, "consensus", types.EventForFinalizer, nil), false)
		}
	}

}

const defaultFinalizeGapHeight = 128

func (f *finalizer) waitFinalizeStartBlock(beginHeight int64) {

	waitHeight := f.chain.cfg.BlockFinalizeGapHeight
	for f.chain.blockStore.Height() < beginHeight+waitHeight {
		time.Sleep(time.Second * 5)
	}

	detail, err := f.chain.GetBlock(beginHeight)
	if err != nil {
		chainlog.Error("waitFinalizeStartBlock", "height", beginHeight, "waitHeight", waitHeight, "get block err", err)
		panic(err)
	}
	_ = f.setFinalizedBlock(detail.GetBlock().Height, detail.GetBlock().Hash(f.chain.client.GetConfig()), false)
	go f.healthCheck(detail.GetBlock().Height)
}

func (f *finalizer) snowmanPreferBlock(msg *queue.Message) {
	//req := (msg.Data).(*types.ReqBytes)
	return

}

func (f *finalizer) snowmanAcceptBlock(msg *queue.Message) {

	req := (msg.Data).(*types.SnowChoice)
	chainlog.Debug("snowmanAcceptBlock", "height", req.Height, "hash", hex.EncodeToString(req.Hash))

	// 已经最终化区块不在当前最佳链中, 即当前节点在侧链上, 最终化记录不更新
	if !f.chain.bestChain.HaveBlock(req.GetHash(), req.GetHeight()) {
		chainlog.Debug("snowmanAcceptBlock not in bestChain", "height", req.Height,
			"hash", hex.EncodeToString(req.GetHash()), "chainHeight", f.chain.bestChain.height())
		return
	}

	err := f.setFinalizedBlock(req.GetHeight(), req.GetHash(), true)
	if err != nil {
		chainlog.Error("snowmanAcceptBlock", "setFinalizedBlock err", err.Error())
	}
}

func (f *finalizer) setFinalizedBlock(height int64, hash []byte, mustInorder bool) error {

	chainlog.Debug("setFinalizedBlock", "height", height, "hash", hex.EncodeToString(hash))
	f.lock.Lock()
	defer f.lock.Unlock()
	if mustInorder && height <= f.choice.Height {
		chainlog.Debug("setFinalizedBlock disorder", "height", height, "currHeight", f.choice.Height)
		return nil
	}
	f.choice.Height = height
	f.choice.Hash = hash
	err := f.chain.blockStore.db.Set(snowChoiceKey, types.Encode(&f.choice))
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
