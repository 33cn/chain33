// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

//var
var (
	tempBlockKey     = []byte("TB:")
	lastTempBlockKey = []byte("LTB:")
)

//const
const (
	//waitTimeDownLoad 节点启动之后等待开始快速下载的时间秒，超时就切换到普通同步模式
	waitTimeDownLoad = 120

	//快速下载时需要的最少peer数量
	bestPeerCount = 2
)

//DownLoadInfo blockchain模块下载block处理结构体
type DownLoadInfo struct {
	StartHeight int64
	EndHeight   int64
	Pids        []string
}

//ErrCountInfo  启动download时read一个block失败等待最长时间为2分钟，120秒
type ErrCountInfo struct {
	Height int64
	Count  int64
}

//存储temp block height 对应的block
func calcHeightToTempBlockKey(height int64) []byte {
	return append(tempBlockKey, []byte(fmt.Sprintf("%012d", height))...)
}

//存储last temp block height
func calcLastTempBlockHeightKey() []byte {
	return lastTempBlockKey
}

//GetDownloadSyncStatus 获取下载区块的同步模式
func (chain *BlockChain) GetDownloadSyncStatus() bool {
	chain.fastDownLoadSynLock.Lock()
	defer chain.fastDownLoadSynLock.Unlock()
	return chain.isFastDownloadSync
}

//UpdateDownloadSyncStatus 更新下载区块的同步模式
func (chain *BlockChain) UpdateDownloadSyncStatus(Sync bool) {
	chain.fastDownLoadSynLock.Lock()
	defer chain.fastDownLoadSynLock.Unlock()
	chain.isFastDownloadSync = Sync
}

//FastDownLoadBlocks 开启快速下载区块的模式
func (chain *BlockChain) FastDownLoadBlocks() {
	curHeight := chain.GetBlockHeight()
	lastTempHight := chain.GetLastTempBlockHeight()

	synlog.Info("FastDownLoadBlocks", "curHeight", curHeight, "lastTempHight", lastTempHight)

	//需要执行完上次已经下载并临时存贮在db中的blocks
	if lastTempHight != -1 && lastTempHight > curHeight {
		chain.ReadBlockToExec(lastTempHight, false)
	}
	//1：满足bestpeer数量，并且落后区块数量大于5000个开启快速同步
	//2：落后区块数量小于5000个不开启快速同步，启动普通同步模式
	//3：启动二分钟如果还不满足快速下载的条件就直接退出，启动普通同步模式

	startTime := types.Now()

	for {
		curheight := chain.GetBlockHeight()
		peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()
		pids := chain.GetBestChainPids()
		//节点启动时只有落后最优链batchsyncblocknum个区块时才开启这种下载模式
		if pids != nil && peerMaxBlkHeight != -1 && curheight+batchsyncblocknum >= peerMaxBlkHeight {
			chain.UpdateDownloadSyncStatus(false)
			synlog.Info("FastDownLoadBlocks:quit!", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight)
			break
		} else if curheight+batchsyncblocknum < peerMaxBlkHeight && len(pids) >= bestPeerCount {
			synlog.Info("start download blocks!FastDownLoadBlocks", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight)
			go chain.ProcDownLoadBlocks(curheight, peerMaxBlkHeight, pids)
			go chain.ReadBlockToExec(peerMaxBlkHeight, true)
			break
		} else if types.Since(startTime) > waitTimeDownLoad*time.Second || chain.cfg.SingleMode {
			chain.UpdateDownloadSyncStatus(false)
			synlog.Info("FastDownLoadBlocks:waitTimeDownLoad:quit!", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight, "pids", pids)
			break
		} else {
			synlog.Info("FastDownLoadBlocks task sleep 1 second !")
			time.Sleep(time.Second)
		}
	}
}

//ReadBlockToExec 执行快速下载临时存储在db中的block
func (chain *BlockChain) ReadBlockToExec(height int64, isNewStart bool) {
	synlog.Info("ReadBlockToExec starting!!!", "height", height, "isNewStart", isNewStart)
	var waitCount ErrCountInfo
	waitCount.Height = 0
	waitCount.Count = 0
	for {
		curheight := chain.GetBlockHeight()
		peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()

		// 节点同步阶段自己高度小于最大高度batchsyncblocknum时存储block到db批量处理时不刷盘
		if peerMaxBlkHeight > curheight+batchsyncblocknum && !chain.cfgBatchSync {
			atomic.CompareAndSwapInt32(&chain.isbatchsync, 1, 0)
		} else {
			atomic.CompareAndSwapInt32(&chain.isbatchsync, 0, 1)
		}
		if (curheight >= peerMaxBlkHeight && peerMaxBlkHeight != -1) || curheight >= height {
			chain.cancelFastDownLoadFlag(isNewStart)
			synlog.Info("ReadBlockToExec complete!", "curheight", curheight, "height", height, "peerMaxBlkHeight", peerMaxBlkHeight)
			break
		}
		block, err := chain.ReadBlockByHeight(curheight + 1)
		if err != nil {
			//在downLoadTask任务退出后，尝试获取block2分钟，还获取不到就直接退出download下载
			if isNewStart {
				if !chain.downLoadTask.InProgress() {
					if waitCount.Height == curheight+1 {
						waitCount.Count++
					} else {
						waitCount.Height = curheight + 1
						waitCount.Count = 1
					}
					if waitCount.Count >= 120 {
						chain.cancelFastDownLoadFlag(isNewStart)
						synlog.Error("ReadBlockToExec:ReadBlockByHeight:timeout", "height", curheight+1, "peerMaxBlkHeight", peerMaxBlkHeight, "err", err)
						break
					}
					time.Sleep(time.Second)
					continue
				} else {
					synlog.Info("ReadBlockToExec:ReadBlockByHeight", "height", curheight+1, "peerMaxBlkHeight", peerMaxBlkHeight, "err", err)
					time.Sleep(time.Second)
					continue
				}
			} else {
				chain.cancelFastDownLoadFlag(isNewStart)
				synlog.Error("ReadBlockToExec:ReadBlockByHeight", "height", curheight+1, "peerMaxBlkHeight", peerMaxBlkHeight, "err", err)
				break
			}
		}
		_, ismain, isorphan, err := chain.ProcessBlock(false, &types.BlockDetail{Block: block}, "download", true, -1)
		if err != nil {
			//执行失败强制结束快速下载模式并切换到普通下载模式
			if isNewStart && chain.downLoadTask.InProgress() {
				Err := chain.downLoadTask.Cancel()
				if Err != nil {
					synlog.Error("ReadBlockToExec:downLoadTask.Cancel!", "height", block.Height, "hash", common.ToHex(block.Hash()), "isNewStart", isNewStart, "err", Err)
				}
				chain.DefaultDownLoadInfo()
			}
			chain.cancelFastDownLoadFlag(isNewStart)
			synlog.Error("ReadBlockToExec:ProcessBlock:err!", "height", block.Height, "hash", common.ToHex(block.Hash()), "isNewStart", isNewStart, "err", err)
			break
		}
		synlog.Debug("ReadBlockToExec:ProcessBlock:success!", "height", block.Height, "ismain", ismain, "isorphan", isorphan, "hash", common.ToHex(block.Hash()))
	}
}

//CancelFastDownLoadFlag 清除快速下载模式的一些标志
func (chain *BlockChain) cancelFastDownLoadFlag(isNewStart bool) {
	if isNewStart {
		chain.UpdateDownloadSyncStatus(false)
	}
	chain.DelLastTempBlockHeight()
	synlog.Info("cancelFastDownLoadFlag", "isNewStart", isNewStart)
}

//ReadBlockByHeight 从数据库中读取快速下载临时存储的block信息
func (chain *BlockChain) ReadBlockByHeight(height int64) (*types.Block, error) {
	blockByte, err := chain.blockStore.db.Get(calcHeightToTempBlockKey(height))
	if blockByte == nil || err != nil {
		return nil, types.ErrHeightNotExist
	}
	var block types.Block
	err = proto.Unmarshal(blockByte, &block)
	if err != nil {
		storeLog.Error("ReadBlockByHeight", "err", err)
		return nil, err
	}
	//读取成功之后将将此临时存贮删除
	err = chain.blockStore.db.Delete(calcHeightToTempBlockKey(height - 1))
	if err != nil {
		storeLog.Error("ReadBlockByHeight:Delete", "height", height, "err", err)
	}
	return &block, err
}

//WriteBlockToDbTemp 快速下载的block临时存贮到数据库
func (chain *BlockChain) WriteBlockToDbTemp(block *types.Block) error {
	if block == nil {
		panic("WriteBlockToDbTemp block is nil")
	}
	sync := true
	if atomic.LoadInt32(&chain.isbatchsync) == 0 {
		sync = false
	}
	beg := types.Now()
	defer func() {
		chainlog.Debug("WriteBlockToDbTemp", "height", block.Height, "sync", sync, "cost", types.Since(beg))
	}()
	newbatch := chain.blockStore.NewBatch(sync)

	blockByte, err := proto.Marshal(block)
	if err != nil {
		chainlog.Error("WriteBlockToDbTemp:Marshal", "height", block.Height)
	}
	newbatch.Set(calcHeightToTempBlockKey(block.Height), blockByte)
	heightbytes := types.Encode(&types.Int64{Data: block.Height})
	newbatch.Set(calcLastTempBlockHeightKey(), heightbytes)
	return newbatch.Write()
}

//GetLastTempBlockHeight 从数据库中获取快速下载的最新的block高度
func (chain *BlockChain) GetLastTempBlockHeight() int64 {
	heightbytes, err := chain.blockStore.db.Get(calcLastTempBlockHeightKey())
	if heightbytes == nil || err != nil {
		chainlog.Error("GetLastTempBlockHeight", "err", err)
		return -1
	}

	var height types.Int64
	err = types.Decode(heightbytes, &height)
	if err != nil {
		chainlog.Error("GetLastTempBlockHeight:Decode", "err", err)
		return -1
	}
	return height.Data
}

//DelLastTempBlockHeight 快速下载结束时删除此标志位
func (chain *BlockChain) DelLastTempBlockHeight() {
	err := chain.blockStore.db.Delete(calcLastTempBlockHeightKey())
	if err != nil {
		synlog.Error("DelLastTempBlockHeight", "err", err)
	}
}

//ProcDownLoadBlocks 处理下载blocks
func (chain *BlockChain) ProcDownLoadBlocks(StartHeight int64, EndHeight int64, pids []string) {
	info := chain.GetDownLoadInfo()

	//可能存在上次DownLoad处理过程中下载区块超时，DownLoad任务退出，但DownLoad没有恢复成默认值
	if info.StartHeight != -1 || info.EndHeight != -1 {
		synlog.Info("ProcDownLoadBlocks", "pids", info.Pids, "StartHeight", info.StartHeight, "EndHeight", info.EndHeight)
	}

	chain.DefaultDownLoadInfo()
	chain.InitDownLoadInfo(StartHeight, EndHeight, pids)
	chain.ReqDownLoadBlocks()
}

//InitDownLoadInfo 开始新的DownLoad处理
func (chain *BlockChain) InitDownLoadInfo(StartHeight int64, EndHeight int64, pids []string) {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()

	chain.downLoadInfo.StartHeight = StartHeight
	chain.downLoadInfo.EndHeight = EndHeight
	chain.downLoadInfo.Pids = pids
	synlog.Debug("InitDownLoadInfo begin", "StartHeight", StartHeight, "EndHeight", EndHeight, "pids", pids)

}

//DefaultDownLoadInfo 将DownLoadInfo恢复成默认值
func (chain *BlockChain) DefaultDownLoadInfo() {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()

	chain.downLoadInfo.StartHeight = -1
	chain.downLoadInfo.EndHeight = -1
	chain.downLoadInfo.Pids = nil
	synlog.Debug("DefaultDownLoadInfo")
}

//GetDownLoadInfo 获取DownLoadInfo
func (chain *BlockChain) GetDownLoadInfo() *DownLoadInfo {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()
	return chain.downLoadInfo
}

//UpdateDownLoadStartHeight 更新DownLoad请求的起始block高度
func (chain *BlockChain) UpdateDownLoadStartHeight(StartHeight int64) {
	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()

	chain.downLoadInfo.StartHeight = StartHeight
	synlog.Debug("UpdateDownLoadStartHeight", "StartHeight", chain.downLoadInfo.StartHeight, "EndHeight", chain.downLoadInfo.EndHeight, "pids", len(chain.downLoadInfo.Pids))
}

//UpdateDownLoadPids 更新bestpeers列表
func (chain *BlockChain) UpdateDownLoadPids() {
	pids := chain.GetBestChainPids()

	chain.downLoadlock.Lock()
	defer chain.downLoadlock.Unlock()
	if len(pids) != 0 {
		chain.downLoadInfo.Pids = pids
		synlog.Info("UpdateDownLoadPids", "StartHeight", chain.downLoadInfo.StartHeight, "EndHeight", chain.downLoadInfo.EndHeight, "pids", len(chain.downLoadInfo.Pids))
	}
}

//ReqDownLoadBlocks 请求DownLoad处理的blocks
func (chain *BlockChain) ReqDownLoadBlocks() {
	info := chain.GetDownLoadInfo()
	if info.StartHeight != -1 && info.EndHeight != -1 && info.Pids != nil {
		synlog.Info("ReqDownLoadBlocks", "StartHeight", info.StartHeight, "EndHeight", info.EndHeight, "pids", len(info.Pids))
		err := chain.FetchBlock(info.StartHeight, info.EndHeight, info.Pids, true)
		if err != nil {
			synlog.Error("ReqDownLoadBlocks:FetchBlock", "err", err)
		}
	}
}
