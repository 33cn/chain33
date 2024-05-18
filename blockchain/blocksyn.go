// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"math/big"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
)

// var
var (
	BackBlockNum            int64 = 128                  //节点高度不增加时向后取blocks的个数
	BackwardBlockNum        int64 = 16                   //本节点高度不增加时并且落后peer的高度数
	checkHeightNoIncSeconds int64 = 200                  //高度不增长时的检测周期
	checkBlockHashSeconds   int64 = 1 * 60               //1分钟检测一次tip hash和peer 对应高度的hash是否一致
	fetchPeerListSeconds    int64 = 5                    //5 秒获取一个peerlist
	MaxRollBlockNum         int64 = 10000                //最大回退block数量
	ReduceHeight                  = MaxRollBlockNum      // 距离最大高度的可精简高度
	SafetyReduceHeight            = ReduceHeight * 3 / 2 // 安全精简高度
	//TODO
	batchsyncblocknum int64 = 5000 //同步阶段，如果自己高度小于最大高度5000个时，saveblock到db时批量处理不刷盘

	synlog = chainlog.New("submodule", "syn")
)

// PeerInfo blockchain模块需要保存的peerinfo
type PeerInfo struct {
	Name       string
	ParentHash []byte
	Height     int64
	Hash       []byte
	choice     *types.SnowChoice
}

// PeerInfoList 节点列表
type PeerInfoList []*PeerInfo

// Len 长度
func (list PeerInfoList) Len() int {
	return len(list)
}

// Less 小于
func (list PeerInfoList) Less(i, j int) bool {
	if list[i].Height < list[j].Height {
		return true
	} else if list[i].Height > list[j].Height {
		return false
	} else {
		return list[i].Name < list[j].Name
	}
}

// Swap 交换
func (list PeerInfoList) Swap(i, j int) {
	temp := list[i]
	list[i] = list[j]
	list[j] = temp
}

// FaultPeerInfo 可疑故障节点信息
type FaultPeerInfo struct {
	Peer        *PeerInfo
	FaultHeight int64
	FaultHash   []byte
	ErrInfo     error
	ReqFlag     bool
}

// BestPeerInfo 用于记录最优链的信息
type BestPeerInfo struct {
	Peer        *PeerInfo
	Height      int64
	Hash        []byte
	Td          *big.Int
	ReqFlag     bool
	IsBestChain bool
}

// BlockOnChain ...
// 记录最新区块上链的时间，长时间没有更新需要做对应的超时处理
// 主要是处理联盟链区块高度相差一个区块
// 整个网络长时间不出块时需要主动去获取最新的区块
// BlockOnChain struct
type BlockOnChain struct {
	sync.RWMutex
	Height      int64
	OnChainTime int64
}

// initOnChainTimeout 初始化
func (chain *BlockChain) initOnChainTimeout() {
	chain.blockOnChain.Lock()
	defer chain.blockOnChain.Unlock()

	chain.blockOnChain.Height = -1
	chain.blockOnChain.OnChainTime = types.Now().Unix()
}

// OnChainTimeout 最新区块长时间没有更新并超过设置的超时时间
func (chain *BlockChain) OnChainTimeout(height int64) bool {
	chain.blockOnChain.Lock()
	defer chain.blockOnChain.Unlock()

	if chain.onChainTimeout == 0 {
		return false
	}

	curTime := types.Now().Unix()
	if chain.blockOnChain.Height != height {
		chain.blockOnChain.Height = height
		chain.blockOnChain.OnChainTime = curTime
		return false
	}
	if curTime-chain.blockOnChain.OnChainTime > chain.onChainTimeout {
		synlog.Debug("OnChainTimeout", "curTime", curTime, "blockOnChain", chain.blockOnChain)
		return true
	}
	return false
}

// SynRoutine 同步事务
func (chain *BlockChain) SynRoutine() {
	//获取peerlist的定时器，默认1分钟
	fetchPeerListTicker := time.NewTicker(time.Duration(fetchPeerListSeconds) * time.Second)
	defer fetchPeerListTicker.Stop()

	//向peer请求同步block的定时器，默认2s
	blockSynTicker := time.NewTicker(chain.blockSynInterVal * time.Second)
	defer blockSynTicker.Stop()
	//5分钟检测一次bestchain主链高度是否有增长，如果没有增长可能是目前主链在侧链上，
	//需要从最高peer向后同步指定的headers用来获取分叉点，再后从指定peer获取分叉点以后的blocks
	checkHeightNoIncreaseTicker := time.NewTicker(time.Duration(checkHeightNoIncSeconds) * time.Second)
	defer checkHeightNoIncreaseTicker.Stop()
	//目前暂定1分钟检测一次本bestchain的tiphash和最高peer的对应高度的blockshash是否一致。
	//如果不一致可能两个节点在各自的链上挖矿，需要从peer的对应高度向后获取指定数量的headers寻找分叉点
	//考虑叉后的第一个block没有广播到本节点，导致接下来广播过来的blocks都是孤儿节点，无法进行主侧链总难度对比
	checkBlockHashTicker := time.NewTicker(time.Duration(checkBlockHashSeconds) * time.Second)

	//5分钟检测一次系统时间，不同步提示告警
	checkClockDriftTicker := time.NewTicker(300 * time.Second)
	if chain.cfg.DisableClockDriftCheck {
		checkBlockHashTicker.Stop()
		checkClockDriftTicker.C = nil
	}
	//3分钟尝试检测一次故障peer是否已经恢复
	recoveryFaultPeerTicker := time.NewTicker(180 * time.Second)
	defer recoveryFaultPeerTicker.Stop()
	//2分钟尝试检测一次最优链，确保本节点在最优链
	checkBestChainTicker := time.NewTicker(120 * time.Second)
	defer checkBestChainTicker.Stop()
	//30s尝试从peer节点请求ChunkRecord
	chunkRecordSynTicker := time.NewTicker(30 * time.Second)
	defer chunkRecordSynTicker.Stop()

	chain.UpdatesynBlkHeight(chain.GetBlockHeight())

	//节点下载模式
	go chain.DownLoadBlocks()

	for {
		select {
		case <-chain.quit:
			//synlog.Info("quit SynRoutine!")
			return
		case <-blockSynTicker.C:
			//synlog.Info("blockSynTicker")
			if chain.GetDownloadSyncStatus() == normalDownLoadMode {
				go chain.SynBlocksFromPeers()
			}

		case <-fetchPeerListTicker.C:
			//synlog.Info("blockUpdateTicker")
			chain.tickerwg.Add(1)
			go chain.FetchPeerList()

		case <-checkHeightNoIncreaseTicker.C:
			//synlog.Info("CheckHeightNoIncrease")
			chain.tickerwg.Add(1)
			go chain.CheckHeightNoIncrease()

		case <-checkBlockHashTicker.C:
			//synlog.Info("checkBlockHashTicker")
			if chain.GetDownloadSyncStatus() != forkChainDetectMode {
				chain.tickerwg.Add(1)
				go chain.CheckTipBlockHash()
			}

			//定时检查系统时间，如果系统时间有问题，那么会有一个报警
		case <-checkClockDriftTicker.C:
			// ntp可能存在一直没有回应的情况导致go线程不退出，暂时不在WaitGroup中处理
			go chain.checkClockDrift()

			//定时检查故障peer，如果执行出错高度的blockhash值有变化，说明故障peer已经纠正
		case <-recoveryFaultPeerTicker.C:
			chain.tickerwg.Add(1)
			go chain.RecoveryFaultPeer()

			//定时检查peerlist中的节点是否在同一条链上，获取同一高度的blockhash来做对比
		case <-checkBestChainTicker.C:
			chain.tickerwg.Add(1)
			go chain.CheckBestChain(false)

			//定时检查ChunkRecord的同步情况
		case <-chunkRecordSynTicker.C:
			if !chain.cfg.DisableShard {
				go chain.ChunkRecordSync()
			}
		}
	}
}

/*
FetchBlock 函数功能：
通过向P2P模块送 EventFetchBlock(types.RequestGetBlock)，向其他节点主动请求区块，
P2P区块收到这个消息后，会向blockchain 模块回复， EventReply。
其他节点如果有这个范围的区块，P2P模块收到其他节点发来的数据，
会发送送EventAddBlocks(types.Blocks) 给 blockchain 模块，
blockchain 模块回复 EventReply
syncOrfork:true fork分叉处理，不需要处理请求block的个数

	:fasle 区块同步处理，一次请求128个block
*/
func (chain *BlockChain) FetchBlock(start int64, end int64, pid []string, syncOrfork bool) (err error) {
	if chain.client == nil {
		synlog.Error("FetchBlock chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	synlog.Debug("FetchBlock input", "StartHeight", start, "EndHeight", end, "pid", pid)
	blockcount := end - start
	if blockcount < 0 {
		return types.ErrStartBigThanEnd
	}
	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.IsDetail = false
	requestblock.Pid = pid

	//同步block一次请求128个
	if blockcount >= chain.MaxFetchBlockNum {
		requestblock.End = start + chain.MaxFetchBlockNum - 1
	} else {
		requestblock.End = end
	}
	var cb func()
	var timeoutcb func(height int64)
	if syncOrfork {
		//还有区块需要请求，挂接钩子回调函数
		if requestblock.End < chain.downLoadInfo.EndHeight {
			cb = func() {
				chain.ReqDownLoadBlocks()
			}
			timeoutcb = func(height int64) {
				chain.DownLoadTimeOutProc(height)
			}
			chain.UpdateDownLoadStartHeight(requestblock.End + 1)
			//快速下载时需要及时更新bestpeer，防止下载侧链的block
			if chain.GetDownloadSyncStatus() == fastDownLoadMode {
				chain.UpdateDownLoadPids()
			}
		} else { // 所有DownLoad block已请求结束，恢复DownLoadInfo为默认值
			chain.DefaultDownLoadInfo()
		}
		err = chain.downLoadTask.Start(requestblock.Start, requestblock.End, cb, timeoutcb)
		if err != nil {
			return err
		}
	} else {
		if chain.GetPeerMaxBlkHeight()-requestblock.End > BackBlockNum {
			cb = func() {
				chain.SynBlocksFromPeers()
			}
		}
		err = chain.syncTask.Start(requestblock.Start, requestblock.End, cb, timeoutcb)
		if err != nil {
			return err
		}
	}

	synlog.Info("FetchBlock", "Start", requestblock.Start, "End", requestblock.End)
	msg := chain.client.NewMessage("p2p", types.EventFetchBlocks, &requestblock)
	Err := chain.client.Send(msg, true)
	if Err != nil {
		synlog.Error("FetchBlock", "client.Send err:", Err)
		return err
	}
	resp, err := chain.client.Wait(msg)
	if err != nil {
		synlog.Error("FetchBlock", "client.Wait err:", err)
		return err
	}
	return resp.Err()
}

// FetchPeerList 从p2p模块获取peerlist，用于获取active链上最新的高度。
// 如果没有收到广播block就主动向p2p模块发送请求
func (chain *BlockChain) FetchPeerList() {
	defer chain.tickerwg.Done()
	err := chain.fetchPeerList()
	if err != nil {
		synlog.Error("FetchPeerList.", "err", err)
	}
}

func (chain *BlockChain) fetchPeerList() error {
	if chain.client == nil {
		synlog.Error("fetchPeerList chain client not bind message queue.")
		return nil
	}
	msg := chain.client.NewMessage("p2p", types.EventPeerInfo, nil)
	Err := chain.client.SendTimeout(msg, true, 30*time.Second)
	if Err != nil {
		synlog.Error("fetchPeerList", "client.Send err:", Err)
		return Err
	}
	resp, err := chain.client.WaitTimeout(msg, 60*time.Second)
	if err != nil {
		synlog.Error("fetchPeerList", "client.Wait err:", err)
		return err
	}

	peerlist, ok := resp.GetData().(*types.PeerList)
	if !ok {
		synlog.Error("fetchPeerList", "peerlist", "is nil")
		return types.ErrNoPeer
	}
	curheigt := chain.GetBlockHeight()

	var peerInfoList PeerInfoList
	for _, peer := range peerlist.GetPeers() {
		//过滤掉自己和小于自己128高度的节点
		if peer == nil || peer.Self || curheigt > peer.Header.Height+128 {
			continue
		}
		var peerInfo PeerInfo
		peerInfo.Name = peer.Name
		peerInfo.ParentHash = peer.Header.ParentHash
		peerInfo.Height = peer.Header.Height
		peerInfo.Hash = peer.Header.Hash
		peerInfo.choice = peer.Finalized
		peerInfoList = append(peerInfoList, &peerInfo)
	}
	//peerlist中没有比自己节点高的就不做处理直接返回
	if len(peerInfoList) == 0 {
		return nil
	}
	//按照height给peer排序从小到大
	sort.Sort(peerInfoList)

	subInfoList := peerInfoList

	chain.peerMaxBlklock.Lock()
	chain.peerList = subInfoList
	chain.peerMaxBlklock.Unlock()

	//获取到peerlist之后，需要判断是否已经发起了最优链的检测。如果没有就触发一次最优链的检测
	if atomic.LoadInt32(&chain.firstcheckbestchain) == 0 {
		synlog.Info("fetchPeerList trigger first CheckBestChain")
		chain.CheckBestChain(true)
	}
	return nil
}

// GetRcvLastCastBlkHeight 存储广播的block最新高度
func (chain *BlockChain) GetRcvLastCastBlkHeight() int64 {
	chain.castlock.Lock()
	defer chain.castlock.Unlock()
	return chain.rcvLastBlockHeight
}

// UpdateRcvCastBlkHeight 更新广播的block最新高度
func (chain *BlockChain) UpdateRcvCastBlkHeight(height int64) {
	chain.castlock.Lock()
	defer chain.castlock.Unlock()
	chain.rcvLastBlockHeight = height
}

// GetsynBlkHeight 存储已经同步到db的block高度
func (chain *BlockChain) GetsynBlkHeight() int64 {
	chain.synBlocklock.Lock()
	defer chain.synBlocklock.Unlock()
	return chain.synBlockHeight
}

// UpdatesynBlkHeight 更新已经同步到db的block高度
func (chain *BlockChain) UpdatesynBlkHeight(height int64) {
	chain.synBlocklock.Lock()
	defer chain.synBlocklock.Unlock()
	chain.synBlockHeight = height
}

// GetPeerMaxBlkHeight 获取peerlist中合法的最新block高度
func (chain *BlockChain) GetPeerMaxBlkHeight() int64 {
	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()

	//获取peerlist中最高的高度，peerlist是已经按照高度排序了的。
	if chain.peerList != nil {
		peerlen := len(chain.peerList)
		for i := peerlen - 1; i >= 0; i-- {
			if chain.peerList[i] != nil {
				ok := chain.IsFaultPeer(chain.peerList[i].Name)
				if !ok {
					return chain.peerList[i].Height
				}
			}
		}
		//没有合法的peer，此时本节点可能在侧链上，返回peerlist中最高的peer尝试矫正
		maxpeer := chain.peerList[peerlen-1]
		if maxpeer != nil {
			synlog.Debug("GetPeerMaxBlkHeight all peers are faultpeer maybe self on Side chain", "pid", maxpeer.Name, "Height", maxpeer.Height, "Hash", common.ToHex(maxpeer.Hash))
			return maxpeer.Height
		}
	}
	return -1
}

// GetPeerInfo 通过peerid获取peerinfo
func (chain *BlockChain) GetPeerInfo(pid string) *PeerInfo {
	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()

	//获取peerinfo
	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			if pid == peer.Name {
				return peer
			}
		}
	}
	return nil
}

// getForkDetectPeer 区块高度次大节点
func (chain *BlockChain) getForkComparePeer() PeerInfo {

	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()

	if chain.peerList.Len() == 0 {
		return PeerInfo{}
	} else if chain.peerList.Len() == 1 {
		return *chain.peerList[0]
	} else {
		return *chain.peerList[chain.peerList.Len()-2]
	}
}

func (chain *BlockChain) getActivePeersByHeight(height int64) []string {
	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()
	peers := make([]string, 0, 8)
	for _, peer := range chain.peerList {
		if peer.Height >= height {
			peers = append(peers, peer.Name)
		}
	}
	return peers
}

// GetMaxPeerInfo 获取peerlist中最高节点的peerinfo
func (chain *BlockChain) GetMaxPeerInfo() *PeerInfo {
	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()

	//获取peerlist中高度最高的peer，peerlist是已经按照高度排序了的。
	if chain.peerList != nil {
		peerlen := len(chain.peerList)
		for i := peerlen - 1; i >= 0; i-- {
			if chain.peerList[i] != nil {
				ok := chain.IsFaultPeer(chain.peerList[i].Name)
				if !ok {
					return chain.peerList[i]
				}
			}
		}
		//没有合法的peer，此时本节点可能在侧链上，返回peerlist中最高的peer尝试矫正
		maxpeer := chain.peerList[peerlen-1]
		if maxpeer != nil {
			synlog.Debug("GetMaxPeerInfo all peers are faultpeer maybe self on Side chain", "pid", maxpeer.Name, "Height", maxpeer.Height, "Hash", common.ToHex(maxpeer.Hash))
			return maxpeer
		}
	}
	return nil
}

// GetPeerCount get peer count
func (chain *BlockChain) GetPeerCount() int {
	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()
	return chain.peerList.Len()
}

// GetPeers 获取所有peers
func (chain *BlockChain) GetPeers() PeerInfoList {
	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()

	//获取peerinfo
	var peers PeerInfoList

	if chain.peerList != nil {
		peers = append(peers, chain.peerList...)
	}
	return peers
}

// GetPeersMap 获取peers的map列表方便查找
func (chain *BlockChain) GetPeersMap() map[string]bool {
	chain.peerMaxBlklock.RLock()
	defer chain.peerMaxBlklock.RUnlock()
	peersmap := make(map[string]bool)

	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			peersmap[peer.Name] = true
		}
	}
	return peersmap
}

// IsFaultPeer 判断指定pid是否在故障faultPeerList中
func (chain *BlockChain) IsFaultPeer(pid string) bool {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	return chain.faultPeerList[pid] != nil
}

// GetFaultPeer 获取指定pid是否在故障faultPeerList中
func (chain *BlockChain) GetFaultPeer(pid string) *FaultPeerInfo {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	return chain.faultPeerList[pid]
}

// RecoveryFaultPeer 尝试恢复故障peer节点，定时从出错的peer获取出错block的头信息。
// 看对应的block是否有更新。有更新就说明故障peer节点已经恢复ok
func (chain *BlockChain) RecoveryFaultPeer() {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	defer chain.tickerwg.Done()

	//循环遍历故障peerlist，尝试检测故障peer是否已经恢复
	for pid, faultpeer := range chain.faultPeerList {
		//需要考虑回退的情况，可能本地节点已经回退，恢复以前的故障节点，获取故障高度的区块的hash是否在本地已经校验通过。校验通过就直接说明故障已经恢复
		//获取本节点指定高度的blockhash做判断，hash相同就确认故障已经恢复
		blockhash, err := chain.blockStore.GetBlockHashByHeight(faultpeer.FaultHeight)
		if err == nil {
			if bytes.Equal(faultpeer.FaultHash, blockhash) {
				synlog.Debug("RecoveryFaultPeer ", "Height", faultpeer.FaultHeight, "FaultHash", common.ToHex(faultpeer.FaultHash), "pid", pid)
				delete(chain.faultPeerList, pid)
				continue
			}
		}

		err = chain.FetchBlockHeaders(faultpeer.FaultHeight, faultpeer.FaultHeight, pid)
		if err == nil {
			chain.faultPeerList[pid].ReqFlag = true
		}
		synlog.Debug("RecoveryFaultPeer", "pid", faultpeer.Peer.Name, "FaultHeight", faultpeer.FaultHeight, "FaultHash", common.ToHex(faultpeer.FaultHash), "Err", faultpeer.ErrInfo)
	}
}

// AddFaultPeer 添加故障节点到故障FaultPeerList中
func (chain *BlockChain) AddFaultPeer(faultpeer *FaultPeerInfo) {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	//此节点已经存在故障peerlist中打印信息
	faultnode := chain.faultPeerList[faultpeer.Peer.Name]
	if faultnode != nil {
		synlog.Debug("AddFaultPeer old", "pid", faultnode.Peer.Name, "FaultHeight", faultnode.FaultHeight, "FaultHash", common.ToHex(faultnode.FaultHash), "Err", faultnode.ErrInfo)
	}
	chain.faultPeerList[faultpeer.Peer.Name] = faultpeer
	synlog.Debug("AddFaultPeer new", "pid", faultpeer.Peer.Name, "FaultHeight", faultpeer.FaultHeight, "FaultHash", common.ToHex(faultpeer.FaultHash), "Err", faultpeer.ErrInfo)
}

// RemoveFaultPeer 此pid对应的故障已经修复，将此pid从故障列表中移除
func (chain *BlockChain) RemoveFaultPeer(pid string) {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()
	synlog.Debug("RemoveFaultPeer", "pid", pid)

	delete(chain.faultPeerList, pid)
}

// UpdateFaultPeer 更新此故障peer的请求标志位
func (chain *BlockChain) UpdateFaultPeer(pid string, reqFlag bool) {
	chain.faultpeerlock.Lock()
	defer chain.faultpeerlock.Unlock()

	faultpeer := chain.faultPeerList[pid]
	if faultpeer != nil {
		faultpeer.ReqFlag = reqFlag
	}
}

// RecordFaultPeer 当blcok执行出错时，记录出错block高度，hash值，以及出错信息和对应的peerid
func (chain *BlockChain) RecordFaultPeer(pid string, height int64, hash []byte, err error) {

	var faultnode FaultPeerInfo

	//通过pid获取peerinfo
	peerinfo := chain.GetPeerInfo(pid)
	if peerinfo == nil {
		synlog.Error("RecordFaultPeerNode GetPeerInfo is nil", "pid", pid)
		return
	}
	faultnode.Peer = peerinfo
	faultnode.FaultHeight = height
	faultnode.FaultHash = hash
	faultnode.ErrInfo = err
	faultnode.ReqFlag = false
	chain.AddFaultPeer(&faultnode)
}

// SynBlocksFromPeers blockSynSeconds时间检测一次本节点的height是否有增长，没有增长就需要通过对端peerlist获取最新高度，发起同步
func (chain *BlockChain) SynBlocksFromPeers() {

	curheight := chain.GetBlockHeight()
	RcvLastCastBlkHeight := chain.GetRcvLastCastBlkHeight()
	peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()

	// 节点同步阶段自己高度小于最大高度batchsyncblocknum时存储block到db批量处理时不刷盘
	if peerMaxBlkHeight > curheight+batchsyncblocknum && !chain.cfgBatchSync {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 1, 0)
	} else if peerMaxBlkHeight >= 0 {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 0, 1)
	}

	//如果任务正常，那么不重复启动任务
	if chain.syncTask.InProgress() {
		synlog.Info("chain syncTask InProgress")
		return
	}
	//如果此时系统正在处理回滚，不启动同步的任务。
	//等分叉回滚处理结束之后再启动同步任务继续同步
	if chain.downLoadTask.InProgress() {
		synlog.Info("chain downLoadTask InProgress")
		return
	}
	//获取peers的最新高度.处理没有收到广播block的情况
	//落后超过2个区块时主动同步区块，落后一个区块时需要判断是否超时
	backWardThanTwo := curheight+1 < peerMaxBlkHeight
	backWardOne := curheight+1 == peerMaxBlkHeight && chain.OnChainTimeout(curheight)

	if backWardThanTwo || backWardOne {
		synlog.Info("SynBlocksFromPeers", "curheight", curheight, "LastCastBlkHeight", RcvLastCastBlkHeight, "peerMaxBlkHeight", peerMaxBlkHeight)
		pids := chain.GetBestChainPids()
		if pids != nil {
			err := chain.FetchBlock(curheight+1, peerMaxBlkHeight, pids, false)
			if err != nil {
				synlog.Error("SynBlocksFromPeers FetchBlock", "err", err)
			}
		} else {
			synlog.Info("SynBlocksFromPeers GetBestChainPids is nil")
		}
	}
}

// Detect if run in fork chain, try to correct
func (chain *BlockChain) forkChainDetection(prevMode int) {

	// restore download mode
	defer chain.UpdateDownloadSyncStatus(prevMode)

	cmpPeer := chain.getForkComparePeer()
	for cmpPeer.Height == 0 {
		select {
		case <-chain.quit:
			return
		default:
			time.Sleep(time.Second * 5)
			cmpPeer = chain.getForkComparePeer()
		}

	}
	localHeight := chain.GetBlockHeight()
	chainlog.Debug("forkDetectInfo", "height", localHeight, "peerHeight", cmpPeer.Height)
	if cmpPeer.Height < localHeight+BackwardBlockNum {
		return
	}

	if err := chain.FetchBlockHeaders(localHeight-BackwardBlockNum, localHeight, cmpPeer.Name); err != nil {
		chainlog.Error("forkDetectFetchHeaders", "err", err)
	}

	// wait for finding fork point
	var forkHeight int64
	select {
	case <-time.After(time.Minute * 2):
		chainlog.Error("forkDetect wait fork point timeout")
		return
	case forkHeight = <-chain.forkPointChan:
	}

	// no forks
	if forkHeight >= localHeight {
		chainlog.Debug("forkDetectNoForks")
		return
	}

	activePeers := chain.getActivePeersByHeight(cmpPeer.Height)
	chainlog.Debug("forkDetect", "forkHeight", forkHeight, "peerCount", len(activePeers))
	if len(activePeers) <= 0 {
		return
	}

	readyDownload := make(chan struct{})

	go func() {
		for chain.downLoadTask.InProgress() {
			time.Sleep(time.Second * 5)
		}
		readyDownload <- struct{}{}
	}()

	select {
	case <-time.After(time.Minute * 5):
		chainlog.Error("forkDetect wait download task timeout")
		chain.downLoadTask.Cancel()
	case <-readyDownload:
	}
	if chain.syncTask.InProgress() {
		chain.syncTask.Cancel()
	}

	if chain.GetBlockHeight() > localHeight {
		chainlog.Info("forkDetectBlkHeightIncreased", "prev", localHeight, "curr", chain.GetBlockHeight())
		return
	}

	// 分叉点在最终化记录之前, 需要重置最终化引擎
	finalized, _ := chain.finalizer.getLastFinalized()
	if forkHeight < finalized {
		chainlog.Debug("forkChainDetection reset finalize engine", "fork", forkHeight, "finalized", finalized)
		forkHash, err := chain.blockStore.GetBlockHashByHeight(forkHeight)
		if err != nil {
			chainlog.Error("forkChainDetection", "height", forkHeight, "GetBlockHashByHeight err", err)
			return
		}
		_ = chain.finalizer.reset(forkHeight, forkHash)
	}

	// 检测到存在分叉后, 以最近高度作为结束高度
	endHeight := cmpPeer.Height - BackBlockNum + 1
	chainlog.Info("forkDetectDownBlk", "localHeight", localHeight, "endHeight", endHeight, "peers", len(activePeers))
	go chain.ProcDownLoadBlocks(forkHeight, endHeight, activePeers)

}

// CheckHeightNoIncrease 在规定时间本链的高度没有增长，但peerlist中最新高度远远高于本节点高度，
// 可能当前链是在分支链上,需从指定最长链的peer向后请求指定数量的blockheader
// 请求bestchain.Height -BackBlockNum -- bestchain.Height的header
// 需要考虑收不到分叉之后的第一个广播block，这样就会导致后面的广播block都在孤儿节点中了。
func (chain *BlockChain) CheckHeightNoIncrease() {
	defer chain.tickerwg.Done()

	//获取当前主链的最新高度
	localHeight := chain.GetBlockHeight()

	//bestchain的tip高度在变化，更新最新的检测高度即可，高度可能在增长或者回退
	if localHeight != chain.GetsynBlkHeight() {
		chain.UpdatesynBlkHeight(localHeight)
		return
	}

	mode := chain.GetDownloadSyncStatus()
	chainlog.Debug("CheckHeightNoIncrease", "localHeight", localHeight, "downMode", mode)
	// 常规同步模式下, 区块高度不变化, 尝试分叉检测
	if mode == normalDownLoadMode {
		chain.UpdateDownloadSyncStatus(forkChainDetectMode)
		chain.forkChainDetection(mode)
	}
}

// FetchBlockHeaders 从指定pid获取start到end之间的headers
func (chain *BlockChain) FetchBlockHeaders(start int64, end int64, pid string) (err error) {
	if start < 0 {
		start = 0
	}
	if chain.client == nil {
		synlog.Error("FetchBlockHeaders chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	chainlog.Debug("FetchBlockHeaders", "StartHeight", start, "EndHeight", end, "pid", pid)

	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.End = end
	requestblock.IsDetail = false
	requestblock.Pid = []string{pid}

	msg := chain.client.NewMessage("p2p", types.EventFetchBlockHeaders, &requestblock)
	Err := chain.client.Send(msg, true)
	if Err != nil {
		synlog.Error("FetchBlockHeaders", "client.Send err:", Err)
		return err
	}
	resp, err := chain.client.Wait(msg)
	if err != nil {
		synlog.Error("FetchBlockHeaders", "client.Wait err:", err)
		return err
	}
	return resp.Err()
}

// ProcBlockHeader 一个block header消息的处理，分tiphash的校验，故障peer的故障block是否恢复的校验
func (chain *BlockChain) ProcBlockHeader(headers *types.Headers, peerid string) error {

	//判断是否是用于检测故障peer而请求的block header
	faultPeer := chain.GetFaultPeer(peerid)
	if faultPeer != nil && faultPeer.ReqFlag && faultPeer.FaultHeight == headers.Items[0].Height {
		//同一高度的block hash有更新，表示故障peer的故障已经恢复，将此peer从故障peerlist中移除
		if !bytes.Equal(headers.Items[0].Hash, faultPeer.FaultHash) {
			chain.RemoveFaultPeer(peerid)
		} else {
			chain.UpdateFaultPeer(peerid, false)
		}
		return nil
	}

	//检测最优链的处理
	bestchainPeer := chain.GetBestChainPeer(peerid)
	if bestchainPeer != nil && bestchainPeer.ReqFlag && headers.Items[0].Height == bestchainPeer.Height {
		chain.CheckBestChainProc(headers, peerid)
		return nil
	}

	// 用于tiphash对比而请求的block header
	height := headers.Items[0].Height
	//获取height高度在本节点的headers信息
	header, err := chain.blockStore.GetBlockHeaderByHeight(height)
	if err != nil {
		return err
	}
	//对应高度hash不相等就向后寻找分叉点
	if !bytes.Equal(headers.Items[0].Hash, header.Hash) {
		synlog.Info("ProcBlockHeader hash no equal", "height", height, "self hash", common.ToHex(header.Hash), "peer hash", common.ToHex(headers.Items[0].Hash))

		err = chain.FetchBlockHeaders(height-BackBlockNum, height, peerid)
		if err != nil {
			synlog.Info("ProcBlockHeader FetchBlockHeaders", "err", err)
		}
	}
	return nil
}

// ProcBlockHeaders 多个headers消息的处理，主要用于寻找分叉节点
func (chain *BlockChain) ProcBlockHeaders(headers *types.Headers, pid string) error {
	var ForkHeight int64 = -1
	var forkhash []byte
	var err error
	count := len(headers.Items)
	tipheight := chain.bestChain.Height()

	//循环找到分叉点
	for i := count - 1; i >= 0; i-- {
		exists := chain.bestChain.HaveBlock(headers.Items[i].Hash, headers.Items[i].Height)
		if exists {
			ForkHeight = headers.Items[i].Height
			forkhash = headers.Items[i].Hash
			break
		}
	}
	if ForkHeight == -1 {
		synlog.Error("ProcBlockHeaders do not find fork point ")
		synlog.Error("ProcBlockHeaders start headerinfo", "height", headers.Items[0].Height, "hash", common.ToHex(headers.Items[0].Hash))
		synlog.Error("ProcBlockHeaders end headerinfo", "height", headers.Items[count-1].Height, "hash", common.ToHex(headers.Items[count-1].Hash))

		//回退5000个block之后不再回退了，直接返回错误
		startheight := headers.Items[0].Height
		if tipheight > startheight && (tipheight-startheight) > MaxRollBlockNum {
			synlog.Error("ProcBlockHeaders Not Roll Back!", "selfheight", tipheight, "RollBackedhieght", startheight)
			return types.ErrNotRollBack
		}
		//继续向后取指定数量的headers
		height := headers.Items[0].Height
		err = chain.FetchBlockHeaders(height-BackBlockNum, height, pid)
		if err != nil {
			synlog.Info("ProcBlockHeaders FetchBlockHeaders", "err", err)
		}
		return types.ErrContinueBack
	}
	synlog.Info("ProcBlockHeaders find fork point", "height", ForkHeight, "hash", common.ToHex(forkhash))

	if chain.GetDownloadSyncStatus() == forkChainDetectMode {
		synlog.Debug("ProcBlockHeaders forkDetect", "forkHeight", ForkHeight)
		select {
		case chain.forkPointChan <- ForkHeight:
		default:
			synlog.Error("ProcBlockHeaders forkDetect channel block")
		}
		return nil
	}

	//获取此pid对应的peer信息，
	peerinfo := chain.GetPeerInfo(pid)
	if peerinfo == nil {
		synlog.Error("ProcBlockHeaders GetPeerInfo is nil", "pid", pid)
		return types.ErrPeerInfoIsNil
	}

	//从分叉节点高度继续请求block，从pid
	peermaxheight := peerinfo.Height

	//启动一个线程在后台获取分叉的blcok
	if chain.downLoadTask.InProgress() {
		synlog.Info("ProcBlockHeaders downLoadTask.InProgress")
		return nil
	}
	//在快速下载block阶段不处理fork的处理
	//如果在普通同步阶段出现了分叉
	//需要暂定同步解决分叉回滚之后再继续开启普通同步
	if chain.GetDownloadSyncStatus() == normalDownLoadMode {
		if chain.syncTask.InProgress() {
			err = chain.syncTask.Cancel()
			synlog.Info("ProcBlockHeaders: cancel syncTask start fork process downLoadTask!", "err", err)
		}
		endHeight := peermaxheight
		if tipheight < peermaxheight {
			endHeight = tipheight + 1
		}
		go chain.ProcDownLoadBlocks(ForkHeight, endHeight, []string{pid})
	}
	return nil
}

// ProcAddBlockHeadersMsg 处理从peer获取的headers消息
func (chain *BlockChain) ProcAddBlockHeadersMsg(headers *types.Headers, pid string) error {
	if headers == nil {
		return types.ErrInvalidParam
	}
	count := len(headers.Items)
	synlog.Debug("ProcAddBlockHeadersMsg", "count", count, "pid", pid)
	if count == 1 {
		return chain.ProcBlockHeader(headers, pid)
	}
	return chain.ProcBlockHeaders(headers, pid)

}

// CheckTipBlockHash 在规定时间本链的高度没有增长，但peerlist中最新高度远远高于本节点高度，
// 可能当前链是在分支链上,需从指定最长链的peer向后请求指定数量的blockheader
// 请求bestchain.Height -BackBlockNum -- bestchain.Height的header
// 需要考虑收不到分叉之后的第一个广播block，这样就会导致后面的广播block都在孤儿节点中了。
func (chain *BlockChain) CheckTipBlockHash() {
	synlog.Debug("CheckTipBlockHash")
	defer chain.tickerwg.Done()

	//获取当前主链的高度
	tipheight := chain.bestChain.Height()
	tiphash := chain.bestChain.Tip().hash
	laststorheight := chain.blockStore.Height()

	if tipheight != laststorheight {
		synlog.Error("CheckTipBlockHash", "tipheight", tipheight, "laststorheight", laststorheight)
		return
	}

	maxpeer := chain.GetMaxPeerInfo()
	if maxpeer == nil {
		synlog.Error("CheckTipBlockHash GetMaxPeerInfo is nil")
		return
	}
	peermaxheight := maxpeer.Height
	pid := maxpeer.Name
	peerhash := maxpeer.Hash
	var Err error
	//和最高的peer做tip block hash的校验
	if peermaxheight > tipheight {
		//从指定peer 请求BackBlockNum个blockheaders
		synlog.Debug("CheckTipBlockHash >", "peermaxheight", peermaxheight, "tipheight", tipheight)
		Err = chain.FetchBlockHeaders(tipheight, tipheight, pid)
	} else if peermaxheight == tipheight {
		// 直接tip block hash比较,如果不相等需要从peer向后去指定的headers，尝试寻找分叉点
		if !bytes.Equal(tiphash, peerhash) {
			synlog.Debug("CheckTipBlockHash ==", "peermaxheight", peermaxheight, "tipheight", tipheight)
			Err = chain.FetchBlockHeaders(tipheight-BackBlockNum, tipheight, pid)
		}
	} else {

		header, err := chain.blockStore.GetBlockHeaderByHeight(peermaxheight)
		if err != nil {
			return
		}
		if !bytes.Equal(header.Hash, peerhash) {
			synlog.Debug("CheckTipBlockHash<!=", "peermaxheight", peermaxheight, "tipheight", tipheight)
			Err = chain.FetchBlockHeaders(peermaxheight-BackBlockNum, peermaxheight, pid)
		}
	}
	if Err != nil {
		synlog.Error("CheckTipBlockHash FetchBlockHeaders", "err", Err)
	}
}

// IsCaughtUp 本节点是否已经追赶上主链高度，追赶上之后通知本节点的共识模块开始挖矿
func (chain *BlockChain) IsCaughtUp() bool {

	height := chain.GetBlockHeight()

	//peerMaxBlklock.Lock()
	//defer peerMaxBlklock.Unlock()
	peers := chain.GetPeers()
	// peer中只有自己节点，没有其他节点
	if peers == nil {
		synlog.Debug("IsCaughtUp has no peers")
		return chain.cfg.SingleMode
	}

	var maxPeerHeight int64 = -1
	peersNo := 0
	for _, peer := range peers {
		if peer != nil && maxPeerHeight < peer.Height {
			ok := chain.IsFaultPeer(peer.Name)
			if !ok {
				maxPeerHeight = peer.Height
			}
		}
		peersNo++
	}

	isCaughtUp := (height > 0 || types.Since(chain.startTime) > 60*time.Second) && (maxPeerHeight == 0 || (height >= maxPeerHeight && maxPeerHeight != -1))

	synlog.Debug("IsCaughtUp", "IsCaughtUp ", isCaughtUp, "height", height, "maxPeerHeight", maxPeerHeight, "peersNo", peersNo)
	return isCaughtUp
}

// GetNtpClockSyncStatus 获取ntp时间是否同步状态
func (chain *BlockChain) GetNtpClockSyncStatus() bool {
	chain.ntpClockSynclock.Lock()
	defer chain.ntpClockSynclock.Unlock()
	return chain.isNtpClockSync
}

// UpdateNtpClockSyncStatus 定时更新ntp时间同步状态
func (chain *BlockChain) UpdateNtpClockSyncStatus(Sync bool) {
	chain.ntpClockSynclock.Lock()
	defer chain.ntpClockSynclock.Unlock()
	chain.isNtpClockSync = Sync
}

// CheckBestChain 定时确保本节点在最优链上,定时向peer请求指定高度的header
func (chain *BlockChain) CheckBestChain(isFirst bool) {
	if !isFirst {
		defer chain.tickerwg.Done()
	}
	peers := chain.GetPeers()
	// peer中只有自己节点，没有其他节点
	if peers == nil {
		synlog.Debug("CheckBestChain has no peers")
		return
	}

	//设置首次检测最优链的标志
	atomic.CompareAndSwapInt32(&chain.firstcheckbestchain, 0, 1)

	tipheight := chain.bestChain.Height()

	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()

	for _, peer := range peers {
		bestpeer := chain.bestChainPeerList[peer.Name]
		if bestpeer != nil {
			bestpeer.Peer = peer
			bestpeer.Height = tipheight
			bestpeer.Hash = nil
			bestpeer.Td = nil
			bestpeer.ReqFlag = true
		} else {
			if peer.Height < tipheight {
				continue
			}
			var newbestpeer BestPeerInfo
			newbestpeer.Peer = peer
			newbestpeer.Height = tipheight
			newbestpeer.Hash = nil
			newbestpeer.Td = nil
			newbestpeer.ReqFlag = true
			newbestpeer.IsBestChain = false
			chain.bestChainPeerList[peer.Name] = &newbestpeer
		}
		synlog.Debug("CheckBestChain FetchBlockHeaders", "height", tipheight, "pid", peer.Name)
		err := chain.FetchBlockHeaders(tipheight, tipheight, peer.Name)
		if err != nil {
			synlog.Error("CheckBestChain FetchBlockHeaders", "height", tipheight, "pid", peer.Name)
		}
	}
}

// GetBestChainPeer 获取最优节点
func (chain *BlockChain) GetBestChainPeer(pid string) *BestPeerInfo {
	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()
	return chain.bestChainPeerList[pid]
}

// isBestChainPeer 指定peer是不是最优链
func (chain *BlockChain) isBestChainPeer(pid string) bool {
	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()
	peer := chain.bestChainPeerList[pid]
	if peer != nil && peer.IsBestChain {
		return true
	}
	return false
}

// GetBestChainPids 定时确保本节点在最优链上,定时向peer请求指定高度的header
func (chain *BlockChain) GetBestChainPids() []string {
	var PeerPids []string
	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()

	peersmap := chain.GetPeersMap()
	for key, value := range chain.bestChainPeerList {
		if !peersmap[value.Peer.Name] {
			delete(chain.bestChainPeerList, value.Peer.Name)
			synlog.Debug("GetBestChainPids:delete", "peer", value.Peer.Name)
			continue
		}
		if value.IsBestChain {
			ok := chain.IsFaultPeer(value.Peer.Name)
			if !ok {
				PeerPids = append(PeerPids, key)
			}
		}
	}
	synlog.Debug("GetBestChainPids", "pids", PeerPids)
	return PeerPids
}

// CheckBestChainProc 检查最优链
func (chain *BlockChain) CheckBestChainProc(headers *types.Headers, pid string) {

	//获取本节点指定高度的blockhash
	blockhash, err := chain.blockStore.GetBlockHashByHeight(headers.Items[0].Height)
	if err != nil {
		synlog.Debug("CheckBestChainProc GetBlockHashByHeight", "Height", headers.Items[0].Height, "err", err)
		return
	}

	chain.bestpeerlock.Lock()
	defer chain.bestpeerlock.Unlock()

	bestchainpeer := chain.bestChainPeerList[pid]
	if bestchainpeer == nil {
		synlog.Debug("CheckBestChainProc bestChainPeerList is nil", "Height", headers.Items[0].Height, "pid", pid)
		return
	}
	//
	if bestchainpeer.Height == headers.Items[0].Height {
		bestchainpeer.Hash = headers.Items[0].Hash
		bestchainpeer.ReqFlag = false
		if bytes.Equal(headers.Items[0].Hash, blockhash) {
			bestchainpeer.IsBestChain = true
			synlog.Debug("CheckBestChainProc IsBestChain ", "Height", headers.Items[0].Height, "pid", pid)
		} else {
			bestchainpeer.IsBestChain = false
			synlog.Debug("CheckBestChainProc NotBestChain", "Height", headers.Items[0].Height, "pid", pid)
		}
	}
}

// ChunkRecordSync 同步chunkrecord
func (chain *BlockChain) ChunkRecordSync() {
	curheight := chain.GetBlockHeight()
	// 记录检测到高度停止增长的次数
	if curheight == atomic.LoadInt64(&chain.lastHeight) {
		// 若 peerList 只有一个本节点说明网络不通，不记录
		if len(chain.GetPeers()) > 1 {
			atomic.AddInt32(&chain.heightNotIncreaseTimes, 1)
		}
	} else {
		atomic.StoreInt64(&chain.lastHeight, curheight)
		atomic.StoreInt32(&chain.heightNotIncreaseTimes, 0)
	}

	peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()
	recvChunk := chain.GetCurRecvChunkNum()
	// re-download chunk hash after 6 times no increasing.
	if atomic.CompareAndSwapInt32(&chain.heightNotIncreaseTimes, 6, 0) {
		recvChunk = curheight/chain.cfg.ChunkblockNum - 1
		chainlog.Info("ChunkRecordSync redownload", "start", recvChunk+1)
	}

	curShouldChunk, _, _ := chain.CalcChunkInfo(curheight)
	targetChunk, _, _ := chain.CalcSafetyChunkInfo(peerMaxBlkHeight)
	if targetChunk < 0 ||
		curShouldChunk >= targetChunk || //说明已同步上来了不需要再进行chunk请求
		recvChunk >= targetChunk {
		return
	}

	//如果任务正常则不重复启动任务
	if chain.chunkRecordTask.InProgress() {
		synlog.Info("chain chunkRecordTask InProgress")
		return
	}

	synlog.Info("ChunkRecordSync", "curheight", curheight, "peerMaxBlkHeight", peerMaxBlkHeight,
		"recvChunk", recvChunk, "curShouldChunk", curShouldChunk)

	pids := chain.GetBestChainPids()
	if pids != nil {
		err := chain.FetchChunkRecords(recvChunk+1, targetChunk, pids)
		if err != nil {
			synlog.Error("ChunkRecordSync FetchChunkRecords", "err", err)
		}
	} else {
		synlog.Info("ChunkRecordSync GetBestChainPids is nil")
	}
}

// FetchChunkRecords 从指定pid获取start到end之间的ChunkRecord,只需要获取存储归档索引 blockHeight--->chunkhash
func (chain *BlockChain) FetchChunkRecords(start int64, end int64, pid []string) (err error) {
	if chain.client == nil {
		synlog.Error("FetchChunkRecords chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	synlog.Debug("FetchChunkRecords", "StartHeight", start, "EndHeight", end, "pid", pid)

	count := end - start
	if count < 0 {
		return types.ErrStartBigThanEnd
	}
	reqRec := &types.ReqChunkRecords{
		Start:    start,
		End:      end,
		IsDetail: false,
		Pid:      pid,
	}
	var cb func()
	if count >= int64(MaxReqChunkRecord) { // 每次请求最大MaxReqChunkRecord个chunk的record
		reqRec.End = reqRec.Start + int64(MaxReqChunkRecord) - 1
		cb = func() {
			chain.ChunkRecordSync()
		}
	}
	// 目前数据量小可在一个节点下载多个chunk记录
	// TODO 后续可以多个节点并发下载
	err = chain.chunkRecordTask.Start(reqRec.Start, reqRec.End, cb, nil)
	if err != nil {
		return err
	}

	msg := chain.client.NewMessage("p2p", types.EventGetChunkRecord, reqRec)
	err = chain.client.Send(msg, false)
	if err != nil {
		synlog.Error("FetchChunkRecords", "client.Send err:", err)
		return err
	}
	return err
}

// FetchChunkBlockRoutine requests 4 chunks in same time
func (chain *BlockChain) FetchChunkBlockRoutine() {
	atomic.StoreInt32(&chain.chunkDownloading, 1)
	defer atomic.StoreInt32(&chain.chunkDownloading, 0)

	chunkStart := chain.GetBlockHeight() + 1
	synlog.Info("FetchChunkBlockRoutine", "chunkStart", chunkStart)
	// 启动4个goroutine下载chunk，该参数为实验参数，少于4个goroutine时下载速度跟不上执行速度
	concurrency := make(chan struct{}, 4)
	for {
		select {
		case <-chain.quit:
			return
		default:
			// chunk下载速度太快，区块执行速度跟不上
			// 避免临时区块占用太多空间，执行落后100w高度时先sleep一下
			if chunkStart > chain.GetBlockHeight()+1e6 {
				time.Sleep(time.Minute)
				continue
			}

			// 节点生成chunk的高度滞后于当前高度
			_, safetyStart, safetyEnd := chain.CalcSafetyChunkInfo(chain.GetPeerMaxBlkHeight())
			if chunkStart > safetyEnd {
				if chain.GetBlockHeight() > safetyStart {
					return
				}
				// 执行速度跟不上下载速度，此时不能直接退出，可能执行一段时间后又有新的chunk需要下载
				time.Sleep(time.Second * 5)
				continue
			}
			chunkNum, _, chunkEnd := chain.CalcChunkInfo(chunkStart)
			chunkHash, err := chain.blockStore.getRecvChunkHash(chunkNum)
			if err != nil {
				time.Sleep(time.Second * 5)
				continue
			}

			// 由于多个chunk同时下载，可能存在前一个chunk没下载到但后续的chunk已经下载好了的情况，若重启节点又会从当前高度依次下载
			// 因此需要判断一下数据库中是否已经存在下载好了的chunk，避免重复下载
			_, err1 := chain.blockStore.db.Get(calcHeightToTempBlockKey(chunkStart))
			_, err2 := chain.blockStore.db.Get(calcHeightToTempBlockKey(chunkEnd))
			if err1 == nil && err2 == nil {
				synlog.Info("FetchChunkBlockRoutine temp block exist", "chunk", chunkNum, "chunkStart", chunkStart, "chunkEnd", chunkEnd)
				chunkStart = chunkEnd + 1
				continue
			}
			concurrency <- struct{}{}
			go chain.mustFetchChunk(concurrency, &types.ChunkInfoMsg{Start: chunkStart, End: chunkEnd, ChunkHash: chunkHash})
			chunkStart = chunkEnd + 1
		}
	}
}

func (chain *BlockChain) mustFetchChunk(ch <-chan struct{}, info *types.ChunkInfoMsg) {
	defer func() {
		<-ch
	}()
	var retryCount int
	start := time.Now()
	for {
		select {
		case <-chain.quit:
			return
		default:
			retryCount++
			synlog.Info("fetchChunk", "Start", info.Start, "End", info.End, "chunkhash", common.ToHex(info.ChunkHash), "retryCount", retryCount)
			if err := chain.fetchChunk(info); err == nil {
				synlog.Info("fetchChunk finished", "Start", info.Start, "End", info.End, "chunkhash", common.ToHex(info.ChunkHash), "retryCount", retryCount, "time cost", time.Since(start))
				return
			}
		}
	}
}

func (chain *BlockChain) fetchChunk(info *types.ChunkInfoMsg) error {
	msg := chain.client.NewMessage("p2p", types.EventGetChunkBlock, info)
	err := chain.client.Send(msg, true)
	if err != nil {
		return err
	}
	msgReply, err := chain.client.Wait(msg)
	if err != nil {
		return err
	}
	if reply, ok := msgReply.GetData().(*types.Reply); ok && reply.IsOk {
		return nil
	}
	return types.ErrNotFound
}
