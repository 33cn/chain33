package blockchain

import (
	"encoding/hex"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

var (
	snowChoiceKey = []byte("blockchain-snowchoice")
)

type finalizer struct {
	chain        *BlockChain
	choice       types.SnowChoice
	lock         sync.RWMutex
	healthNotify chan struct{}
	resetRunning atomic.Bool
}

func (f *finalizer) Init(chain *BlockChain) {

	f.healthNotify = make(chan struct{}, 1)
	f.chain = chain
	raw, err := chain.blockStore.db.Get(snowChoiceKey)

	if err == nil {
		err = types.Decode(raw, &f.choice)
		if err != nil {
			chainlog.Error("newFinalizer", "decode err", err)
			panic(err)
		}
		chainlog.Info("newFinalizer", "height", f.choice.Height, "hash", hex.EncodeToString(f.choice.Hash))
		go f.healthCheck()
	} else if chain.client.GetConfig().GetModuleConfig().Consensus.Finalizer != "" {
		f.choice.Height = chain.cfg.BlockFinalizeEnableHeight
		chainlog.Info("newFinalizer", "enableHeight", f.choice.Height, "gapHeight", chain.cfg.BlockFinalizeGapHeight)
		go f.waitFinalizeStartBlock(f.choice.Height)
	}
	go f.lazyStart(chain.cfg.BlockFinalizeGapHeight, MaxRollBlockNum)
}

// 基于最大区块回滚深度, 快速收敛, 主要针对同步节点, 减少历史数据共识流程
func (f *finalizer) lazyStart(gapHeight, maxRollbackNum int64) {

	ticker := time.NewTicker(time.Minute * 2)
	minPeerCount := 10
	defer ticker.Stop()
	for {
		select {

		case <-f.chain.quit:
			return
		case <-ticker.C:
			finalized, _ := f.getLastFinalized()
			peerNum := f.chain.GetPeerCount()
			height := f.chain.GetBlockHeight() - gapHeight
			// 连接节点过少||未达到使能高度, 等待连接及区块同步
			if finalized >= height || peerNum < minPeerCount {
				chainlog.Debug("lazyStart wait", "peerNum", peerNum,
					"finalized", finalized, "height", height)
				continue
			}

			maxPeerHeight := f.chain.GetPeerMaxBlkHeight()
			// 已经最终化高度在回滚范围内, 无需加速
			if finalized+maxRollbackNum > maxPeerHeight {
				chainlog.Debug("lazyStart return", "peerNum", peerNum, "finalized", finalized,
					"maxHeight", maxPeerHeight)
				return
			}

			peers := f.chain.getActivePeersByHeight(height + maxRollbackNum)
			chainlog.Debug("lazyStart peer", "peerNum", peerNum, "peersLen", len(peers),
				"finalized", finalized, "height", height)
			// 超过半数节点高度超过该高度, 说明当前链回滚最低高度大于height, 可设为已最终化高度
			if len(peers) < peerNum/2 {
				continue
			}

			detail, err := f.chain.GetBlock(height)
			if err != nil {
				chainlog.Error("lazyStart err", "height", height, "get block err", err)
				continue
			}
			_ = f.reset(height, detail.GetBlock().Hash(f.chain.client.GetConfig()))
		}
	}

}

func (f *finalizer) healthCheck() {

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	healthy := false
	for {
		select {

		case <-f.chain.quit:
			return
		case <-f.healthNotify:
			healthy = true
		case <-ticker.C:
			maxPeerHeight := f.chain.GetPeerMaxBlkHeight()
			// 节点高度落后较多情况不处理, 等待同步
			height := f.chain.GetBlockHeight()
			if height < maxPeerHeight-128 || healthy {
				chainlog.Debug("healthCheck not sync", "healthy", healthy, "height", height, "maxHeight", maxPeerHeight)
				healthy = false
				continue
			}
			finalized, hash := f.getLastFinalized()
			chainlog.Debug("healthCheck timeout", "lastFinalize", finalized,
				"hash", hex.EncodeToString(hash), "chainHeight", height)
			if finalized >= height {
				continue
			}
			// 重新设置高度, 哈希值
			detail, err := f.chain.GetBlock(finalized)
			if err != nil {
				chainlog.Error("finalizer tiemout", "height", finalized, "get block err", err)
				continue
			}
			_ = f.reset(finalized, detail.GetBlock().Hash(f.chain.client.GetConfig()))
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
	go f.healthCheck()
	_ = f.setFinalizedBlock(detail.GetBlock().Height, detail.GetBlock().Hash(f.chain.client.GetConfig()), false)

}

func (f *finalizer) snowmanPreferBlock(msg *queue.Message) {
	//req := (msg.Data).(*types.ReqBytes)
	return

}

func (f *finalizer) snowmanAcceptBlock(msg *queue.Message) {

	req := (msg.Data).(*types.SnowChoice)
	chainlog.Debug("snowmanAcceptBlock", "height", req.Height, "hash", hex.EncodeToString(req.Hash))

	// 已经最终化区块不在当前最佳链中, 即可能在侧链上, 最终化记录不更新
	if !f.chain.bestChain.HaveBlock(req.GetHash(), req.GetHeight()) {
		chainHeight := f.chain.bestChain.Height()
		chainlog.Debug("snowmanAcceptBlock not in bestChain", "height", req.Height,
			"hash", hex.EncodeToString(req.GetHash()), "chainHeight", chainHeight)
		if f.resetRunning.CompareAndSwap(false, true) {
			go f.resetEngine(chainHeight, req, time.Second*10)
		}
		return
	}

	err := f.setFinalizedBlock(req.GetHeight(), req.GetHash(), true)
	if err == nil {
		f.healthNotify <- struct{}{}
	}
}

const consensusTopic = "consensus"

func (f *finalizer) resetEngine(chainHeight int64, sc *types.SnowChoice, duration time.Duration) {

	defer f.resetRunning.Store(false)
	ticker := time.NewTicker(duration)
	defer ticker.Stop()
	for {

		select {

		case <-f.chain.quit:
			return

		case <-ticker.C:

			currHeight := f.chain.bestChain.Height()
			if f.chain.bestChain.HaveBlock(sc.GetHash(), sc.GetHeight()) {
				chainlog.Debug("resetEngine accept", "chainHeight", chainHeight,
					"currHeight", currHeight, "sc.height", sc.GetHeight(), "sc.hash", hex.EncodeToString(sc.GetHash()))
				return
			}
			// 最终化区块不在主链上且主链高度正常增长, 重置最终化引擎, 尝试对该高度重新共识
			if currHeight > chainHeight && currHeight > sc.GetHeight()+12 {
				chainlog.Debug("resetEngine reject", "chainHeight", chainHeight,
					"currHeight", currHeight, "sc.height", sc.GetHeight(), "sc.hash", hex.EncodeToString(sc.GetHash()))
				_ = f.chain.client.Send(queue.NewMessage(types.EventSnowmanResetEngine, consensusTopic, types.EventForFinalizer, nil), true)
				return
			}
		}
	}
}

func (f *finalizer) reset(height int64, hash []byte) error {

	chainlog.Debug("finalizer reset", "height", height, "hash", hex.EncodeToString(hash))
	err := f.setFinalizedBlock(height, hash, false)
	if err != nil {
		chainlog.Error("finalizer reset", "setFinalizedBlock err", err)
		return err
	}
	err = f.chain.client.Send(queue.NewMessage(types.EventSnowmanResetEngine, consensusTopic, types.EventForFinalizer, nil), true)
	if err != nil {
		chainlog.Error("finalizer reset", "send msg err", err)
	}
	return err
}

func (f *finalizer) setFinalizedBlock(height int64, hash []byte, mustInorder bool) error {

	chainlog.Debug("setFinalizedBlock", "height", height, "hash", hex.EncodeToString(hash))
	f.lock.Lock()
	defer f.lock.Unlock()
	if mustInorder && height <= f.choice.Height {
		chainlog.Debug("setFinalizedBlock disorder", "height", height, "currHeight", f.choice.Height)
		return types.ErrInvalidParam
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
	//chainlog.Debug("snowmanLastChoice", "height", height, "hash", hex.EncodeToString(hash))
	msg.Reply(f.chain.client.NewMessage(msg.Topic,
		types.EventSnowmanLastChoice, &types.SnowChoice{Height: height, Hash: hash}))
}
