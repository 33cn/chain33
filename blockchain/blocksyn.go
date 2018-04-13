package blockchain

import (
	"bytes"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	synBlocklock     sync.Mutex
	peerMaxBlklock   sync.Mutex
	castlock         sync.Mutex
	ntpClockSynclock sync.Mutex

	isNtpClockSync = true //ntp时间是否同步

	MaxFetchBlockNum        int64 = 128 * 6 //一次最多申请获取block个数
	TimeoutSeconds          int64 = 2
	BackBlockNum            int64 = 128    //节点高度不增加时向后取blocks的个数
	BackwardBlockNum        int64 = 16     //本节点高度不增加时并且落后peer的高度数
	checkHeightNoIncSeconds int64 = 5 * 60 //高度不增长时的检测周期目前暂定5分钟
	checkBlockHashSeconds   int64 = 1 * 60 //1分钟检测一次tip hash和peer 对应高度的hash是否一致
	fetchPeerListSeconds    int64 = 5      //5 秒获取一个peerlist
	MaxRollBlockNum         int64 = 5000   //最大回退block数量
	//blockSynInterVal              = time.Duration(TimeoutSeconds)
	checkBlockNum           int64 = 128
	batchsyncblocknum       int64 = 5000 //同步阶段，如果自己高度小于最大高度5000个时，saveblock到db时批量处理不刷盘

	synlog = chainlog.New("submodule", "syn")
)

//blockchain模块需要保存的peerinfo
type PeerInfo struct {
	Name       string
	ParentHash []byte
	Height     int64
	Hash       []byte
}
type PeerInfoList []*PeerInfo

func (list PeerInfoList) Len() int {
	return len(list)
}

func (list PeerInfoList) Less(i, j int) bool {
	if list[i].Height < list[j].Height {
		return true
	} else if list[i].Height > list[j].Height {
		return false
	} else {
		return list[i].Name < list[j].Name
	}
}

func (list PeerInfoList) Swap(i, j int) {
	temp := list[i]
	list[i] = list[j]
	list[j] = temp
}

//把peer高度相近的组成一组，目前暂定相差5个高度为一组
type PeerGroup struct {
	PeerCount  int
	StartIndex int
	EndIndex   int
}

func (chain *BlockChain) SynRoutine() {
	//获取peerlist的定时器，默认1分钟
	fetchPeerListTicker := time.NewTicker(time.Duration(fetchPeerListSeconds) * time.Second)

	//向peer请求同步block的定时器，默认2s
	blockSynTicker := time.NewTicker(time.Duration(TimeoutSeconds) * time.Second)

	//5分钟检测一次bestchain主链高度是否有增长，如果没有增长可能是目前主链在侧链上，
	//需要从最高peer向后同步指定的headers用来获取分叉点，再后从指定peer获取分叉点以后的blocks
	checkHeightNoIncreaseTicker := time.NewTicker(time.Duration(checkHeightNoIncSeconds) * time.Second)

	//目前暂定1分钟检测一次本bestchain的tiphash和最高peer的对应高度的blockshash是否一致。
	//如果不一致可能两个节点在各自的链上挖矿，需要从peer的对应高度向后获取指定数量的headers寻找分叉点
	//考虑叉后的第一个block没有广播到本节点，导致接下来广播过来的blocks都是孤儿节点，无法进行主侧链总难度对比
	checkBlockHashTicker := time.NewTicker(time.Duration(checkBlockHashSeconds) * time.Second)

	//5分钟检测一次系统时间，不同步提示告警
	checkClockDriftTicker := time.NewTicker(300 * time.Second)

	for {
		select {
		case <-chain.quit:
			//synlog.Info("quit poolRoutine!")
			return
		case _ = <-blockSynTicker.C:
			//synlog.Info("blockSynTicker")
			chain.SynBlocksFromPeers()

		case _ = <-fetchPeerListTicker.C:
			//synlog.Info("blockUpdateTicker")
			chain.FetchPeerList()

		case _ = <-checkHeightNoIncreaseTicker.C:
			//synlog.Info("CheckHeightNoIncrease")
			chain.CheckHeightNoIncrease()

		case _ = <-checkBlockHashTicker.C:
			//synlog.Info("checkBlockHashTicker")
			chain.CheckTipBlockHash()
		//定时检查系统时间，如果系统时间有问题，那么会有一个报警
		case _ = <-checkClockDriftTicker.C:
			checkClockDrift()
		}
	}
}

/*
函数功能：
通过向P2P模块送 EventFetchBlock(types.RequestGetBlock)，向其他节点主动请求区块，
P2P区块收到这个消息后，会向blockchain 模块回复， EventReply。
其他节点如果有这个范围的区块，P2P模块收到其他节点发来的数据，
会发送送EventAddBlocks(types.Blocks) 给 blockchain 模块，
blockchain 模块回复 EventReply
结构体：
*/
func (chain *BlockChain) FetchBlock(start int64, end int64, pid []string) (err error) {
	if chain.client == nil {
		synlog.Error("FetchBlock chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	synlog.Debug("FetchBlock input", "StartHeight", start, "EndHeight", end, "pid", pid[0])
	blockcount := end - start
	if blockcount < 0 {
		return types.ErrStartBigThanEnd
	}
	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.Isdetail = false
	requestblock.Pid = pid

	if blockcount >= MaxFetchBlockNum {
		requestblock.End = start + MaxFetchBlockNum - 1
	} else {
		requestblock.End = end
	}
	var cb func()
	if chain.GetPeerMaxBlkHeight()-requestblock.End > BackBlockNum {
		cb = func() {
			chain.SynBlocksFromPeers()
		}
	}
	err = chain.task.Start(requestblock.Start, requestblock.End, cb)
	if err != nil {
		return err
	}
	synlog.Debug("FetchBlock", "Start", requestblock.Start, "End", requestblock.End)
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

//从p2p模块获取peerlist，用于获取active链上最新的高度。
//如果没有收到广播block就主动向p2p模块发送请求
func (chain *BlockChain) FetchPeerList() {
	chain.fetchPeerList()
}

var debugflag = 60

func (chain *BlockChain) fetchPeerList() error {
	if chain.client == nil {
		synlog.Error("fetchPeerList chain client not bind message queue.")
		return nil
	}
	msg := chain.client.NewMessage("p2p", types.EventPeerInfo, nil)
	Err := chain.client.Send(msg, true)
	if Err != nil {
		synlog.Error("fetchPeerList", "client.Send err:", Err)
		return Err
	}
	resp, err := chain.client.Wait(msg)
	if err != nil {
		synlog.Error("fetchPeerList", "client.Wait err:", err)
		return err
	}

	peerlist := resp.GetData().(*types.PeerList)
	if peerlist == nil {
		synlog.Error("fetchPeerList", "peerlist", "is nil")
		return types.ErrNoPeer
	}
	curheigt := chain.GetBlockHeight()

	var peerInfoList PeerInfoList
	for _, peer := range peerlist.Peers {
		//chainlog.Info("fetchPeerList", "peername:", peer.Name, "peerHeight:", peer.Header.Height)

		//过滤掉自己和小于自己的节点
		if peer.Self || curheigt > peer.Header.Height+5 {
			continue
		}
		var peerInfo PeerInfo
		peerInfo.Name = peer.Name
		peerInfo.ParentHash = peer.Header.ParentHash
		peerInfo.Height = peer.Header.Height
		peerInfo.Hash = peer.Header.Hash
		peerInfoList = append(peerInfoList, &peerInfo)
	}
	//peerlist中没有比自己节点高的就不做处理直接返回
	if len(peerInfoList) == 0 {
		return nil
	}
	//按照height给peer排序从小到大
	sort.Sort(peerInfoList)

	subInfoList := maxSubList(peerInfoList)

	//debug
	debugflag++
	if debugflag >= 60 {
		for _, peerinfo := range subInfoList {
			synlog.Debug("fetchPeerList subInfoList", "Name", peerinfo.Name, "Height", peerinfo.Height, "ParentHash", common.ToHex(peerinfo.ParentHash), "Hash", common.ToHex(peerinfo.Hash))
		}
		debugflag = 0
	}
	peerMaxBlklock.Lock()
	chain.peerList = subInfoList
	peerMaxBlklock.Unlock()

	return nil
}

func maxSubList(list PeerInfoList) (sub PeerInfoList) {
	start := 0
	end := 0
	if len(list) == 0 {
		return list
	}
	for i := 0; i < len(list); i++ {
		var nextheight int64
		if i+1 == len(list) {
			nextheight = math.MaxInt64
		} else {
			nextheight = list[i+1].Height
		}
		if nextheight-list[i].Height > checkBlockNum {
			end = i + 1
			if len(sub) < (end - start) {
				sub = list[start:end]
			}
			start = i + 1
			end = i + 1
		} else {
			end = i + 1
		}
	}
	//只有一个节点，那么取最高的节点
	if len(sub) <= 1 {
		return list[len(list)-1:]
	}
	return sub
}

//存储广播的block最新高度
func (chain *BlockChain) GetRcvLastCastBlkHeight() int64 {
	castlock.Lock()
	defer castlock.Unlock()
	return chain.rcvLastBlockHeight
}

func (chain *BlockChain) UpdateRcvCastBlkHeight(height int64) {
	castlock.Lock()
	defer castlock.Unlock()
	chain.rcvLastBlockHeight = height
}

//存储已经同步到db的block高度
func (chain *BlockChain) GetsynBlkHeight() int64 {
	synBlocklock.Lock()
	defer synBlocklock.Unlock()
	return chain.synBlockHeight
}

func (chain *BlockChain) UpdatesynBlkHeight(height int64) {
	synBlocklock.Lock()
	defer synBlocklock.Unlock()
	chain.synBlockHeight = height
}

// 获取peer的最新block高度
func (chain *BlockChain) GetPeerMaxBlkHeight() int64 {
	peerMaxBlklock.Lock()
	defer peerMaxBlklock.Unlock()

	//获取peerlist中最高的高度
	var maxPeerHeight int64 = -1
	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			if peer != nil && maxPeerHeight < peer.Height {
				maxPeerHeight = peer.Height
			}
		}
	}
	return maxPeerHeight

}

func (chain *BlockChain) GetPeerMaxBlkPid() string {
	peerMaxBlklock.Lock()
	defer peerMaxBlklock.Unlock()

	//获取peerlist中最高高度的pid
	var maxPeerHeight int64 = -1
	var pid string
	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			if peer != nil && maxPeerHeight < peer.Height {
				pid = peer.Name
			}
		}
	}
	return pid
}

func (chain *BlockChain) GetPeerMaxBlkHash() []byte {
	peerMaxBlklock.Lock()
	defer peerMaxBlklock.Unlock()

	//获取peerlist中最高高度的blockhash
	var maxPeerHeight int64 = -1
	hash := common.Hash{}.Bytes()

	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			if peer != nil && maxPeerHeight < peer.Height {
				hash = peer.Hash
			}
		}
	}
	return hash
}

func (chain *BlockChain) GetPeerPids() []string {
	peerMaxBlklock.Lock()
	defer peerMaxBlklock.Unlock()

	//获取peerlist中最高的高度
	var PeerPids []string
	if chain.peerList != nil {
		for _, peer := range chain.peerList {
			PeerPids = append(PeerPids, peer.Name)
		}
	}
	return PeerPids
}

//blockSynSeconds时间检测一次本节点的height是否有增长，没有增长就需要通过对端peerlist获取最新高度，发起同步
func (chain *BlockChain) SynBlocksFromPeers() {

	curheight := chain.GetBlockHeight()
	RcvLastCastBlkHeight := chain.GetRcvLastCastBlkHeight()
	peerMaxBlkHeight := chain.GetPeerMaxBlkHeight()

	// 节点同步阶段自己高度小于最大高度batchsyncblocknum时存储block到db批量处理时不刷盘
	if peerMaxBlkHeight > curheight+batchsyncblocknum && !chain.cfgBatchSync {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 1, 0)
	} else {
		atomic.CompareAndSwapInt32(&chain.isbatchsync, 0, 1)
	}
	//synlog.Info("SynBlocksFromPeers", "isbatchsync", chain.isbatchsync)

	//如果任务正常，那么不重复启动任务
	if chain.task.InProgress() {
		synlog.Info("chain task InProgress")
		return
	}
	//获取peers的最新高度.处理没有收到广播block的情况
	if curheight+1 < peerMaxBlkHeight {
		synlog.Info("SynBlocksFromPeers", "curheight", curheight, "LastCastBlkHeight", RcvLastCastBlkHeight, "peerMaxBlkHeight", peerMaxBlkHeight)
		chain.FetchBlock(curheight+1, peerMaxBlkHeight, chain.GetPeerPids())
	}

	return
}

//在规定时间本链的高度没有增长，但peerlist中最新高度远远高于本节点高度，
//可能当前链是在分支链上,需从指定最长链的peer向后请求指定数量的blockheader
//请求bestchain.Height -BackBlockNum -- bestchain.Height的header
//需要考虑收不到分叉之后的第一个广播block，这样就会导致后面的广播block都在孤儿节点中了。
func (chain *BlockChain) CheckHeightNoIncrease() {
	synlog.Debug("CheckHeightNoIncrease")

	//获取当前主链的最新高度
	tipheight := chain.bestChain.Height()
	laststorheight := chain.blockStore.Height()

	if tipheight != laststorheight {
		synlog.Error("CheckHeightNoIncrease", "tipheight", tipheight, "laststorheight", laststorheight)
		return
	}
	//获取上个检测周期时的检测高度
	checkheight := chain.GetsynBlkHeight()

	//bestchain的tip高度在变化，更新最新的检测高度即可，高度可能在增长或者回退
	if tipheight != checkheight {
		chain.UpdatesynBlkHeight(tipheight)
		return
	}
	//一个检测周期bestchain的tip高度没有变化。并且远远落后于peer的最新高度
	//本节点可能在侧链上，需要从最新的peer上向后取BackBlockNum个headers
	peermaxheight := chain.GetPeerMaxBlkHeight()
	pid := chain.GetPeerMaxBlkPid()
	if peermaxheight > tipheight && (peermaxheight-tipheight) > BackwardBlockNum {
		//从指定peer 请求BackBlockNum个blockheaders
		if tipheight > BackBlockNum {
			chain.FetchBlockHeaders(tipheight-BackBlockNum, tipheight, pid)
		} else {
			chain.FetchBlockHeaders(0, tipheight, pid)
		}
	}
	return
}

//从指定pid获取start到end之间的headers
func (chain *BlockChain) FetchBlockHeaders(start int64, end int64, pid string) (err error) {
	if chain.client == nil {
		synlog.Error("FetchBlockHeaders chain client not bind message queue.")
		return types.ErrClientNotBindQueue
	}

	chainlog.Debug("FetchBlockHeaders", "StartHeight", start, "EndHeight", end, "pid", pid)

	var requestblock types.ReqBlocks
	requestblock.Start = start
	requestblock.End = end
	requestblock.Isdetail = false
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

//处理从peer获取的headers消息
func (chain *BlockChain) ProcAddBlockHeadersMsg(headers *types.Headers) error {
	if headers == nil {
		return types.ErrInputPara
	}
	count := len(headers.Items)
	synlog.Debug("ProcAddBlockHeadersMsg", "headers count", count)
	// 处理tiphash对比的操作
	if count == 1 {
		height := headers.Items[0].Height
		//获取height高度在本节点的headers信息
		header, err := chain.blockStore.GetBlockHeaderByHeight(height)
		if err != nil {
			return err
		}
		//对应高度hash不相等就向后寻找分叉点
		pid := chain.GetPeerMaxBlkPid()
		if !bytes.Equal(headers.Items[0].Hash, header.Hash) {
			synlog.Info("ProcAddBlockHeadersMsg hash no equal", "height", height, "self hash", common.ToHex(header.Hash), "peer hash", common.ToHex(headers.Items[0].Hash))

			if height > BackBlockNum {
				chain.FetchBlockHeaders(height-BackBlockNum, height, pid)
			} else if height != 0 {
				chain.FetchBlockHeaders(0, height, pid)
			}
		}

		return nil
	}
	var ForkHeight int64 = -1
	var forkhash []byte
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
		synlog.Error("ProcAddBlockHeadersMsg do not find fork point ")
		synlog.Error("ProcAddBlockHeadersMsg start headerinfo", "height", headers.Items[0].Height, "hash", common.ToHex(headers.Items[0].Hash))
		synlog.Error("ProcAddBlockHeadersMsg end headerinfo", "height", headers.Items[count-1].Height, "hash", common.ToHex(headers.Items[count-1].Hash))

		//回退5000个block之后不再回退了，直接返回错误
		tipheight := chain.bestChain.Height()
		startheight := headers.Items[0].Height
		if tipheight > startheight && (tipheight-startheight) > MaxRollBlockNum {
			synlog.Error("ProcAddBlockHeadersMsg Not Roll Back!", "selfheight", tipheight, "RollBackedhieght", startheight)
			return types.ErrNotRollBack
		}
		//继续向后取指定数量的headers
		height := headers.Items[0].Height
		pid := chain.GetPeerMaxBlkPid()
		if height > BackBlockNum {
			chain.FetchBlockHeaders(height-BackBlockNum, height, pid)
		} else {
			chain.FetchBlockHeaders(0, height, pid)
		}

		return types.ErrContinueBack
	}
	synlog.Info("ProcAddBlockHeadersMsg find fork point", "height", ForkHeight, "hash", common.ToHex(forkhash))

	//从分叉节点高度继续请求block，从pid
	pid := chain.GetPeerMaxBlkPid()
	peermaxheight := chain.GetPeerMaxBlkHeight()
	//此时停止同步的任务
	chain.task.Cancel()
	if peermaxheight > ForkHeight+MaxFetchBlockNum {
		chain.FetchBlock(ForkHeight, ForkHeight+MaxFetchBlockNum, []string{pid})
	} else {
		chain.FetchBlock(ForkHeight, peermaxheight, []string{pid})
	}
	return nil
}

//在规定时间本链的高度没有增长，但peerlist中最新高度远远高于本节点高度，
//可能当前链是在分支链上,需从指定最长链的peer向后请求指定数量的blockheader
//请求bestchain.Height -BackBlockNum -- bestchain.Height的header
//需要考虑收不到分叉之后的第一个广播block，这样就会导致后面的广播block都在孤儿节点中了。
func (chain *BlockChain) CheckTipBlockHash() {
	synlog.Debug("CheckTipBlockHash")

	//获取当前主链的高度
	tipheight := chain.bestChain.Height()
	tiphash := chain.bestChain.tip().hash
	laststorheight := chain.blockStore.Height()

	if tipheight != laststorheight {
		synlog.Error("CheckTipBlockHash", "tipheight", tipheight, "laststorheight", laststorheight)
		return
	}

	peermaxheight := chain.GetPeerMaxBlkHeight()
	pid := chain.GetPeerMaxBlkPid()
	peerhash := chain.GetPeerMaxBlkHash()
	if peermaxheight > tipheight {
		//从指定peer 请求BackBlockNum个blockheaders
		synlog.Debug("CheckTipBlockHash >", "start", tipheight, "end", tipheight)
		synlog.Debug("CheckTipBlockHash >", "peermaxheight", peermaxheight, "tipheight", tipheight)

		chain.FetchBlockHeaders(tipheight, tipheight, pid)
	} else if peermaxheight == tipheight {
		// 直接tip block hash比较,如果不相等需要从peer向后去指定的headers，尝试寻找分叉点
		if !bytes.Equal(tiphash, peerhash) {
			if tipheight > BackBlockNum {
				synlog.Debug("CheckTipBlockHash ==", "start", tipheight-BackBlockNum, "end", tipheight)
				synlog.Debug("CheckTipBlockHash ==", "peermaxheight", peermaxheight, "tipheight", tipheight)

				chain.FetchBlockHeaders(tipheight-BackBlockNum, tipheight, pid)
			} else {
				synlog.Debug("CheckTipBlockHash !=", "start", "1", "end", tipheight)
				synlog.Debug("CheckTipBlockHash !=", "peermaxheight", peermaxheight, "tipheight", tipheight)

				chain.FetchBlockHeaders(1, tipheight, pid)
			}
		}
	} else {
		header, err := chain.blockStore.GetBlockHeaderByHeight(peermaxheight)
		if err != nil {
			return
		}
		if !bytes.Equal(header.Hash, peerhash) {
			if peermaxheight > BackBlockNum {
				synlog.Debug("CheckTipBlockHash <!=", "start", peermaxheight-BackBlockNum, "end", tipheight)
				synlog.Debug("CheckTipBlockHash<!=", "peermaxheight", peermaxheight, "tipheight", tipheight)

				chain.FetchBlockHeaders(peermaxheight-BackBlockNum, peermaxheight, pid)
			} else {
				synlog.Debug("CheckTipBlockHash<!=", "start", "1", "end", tipheight)
				synlog.Debug("CheckTipBlockHash<!=", "peermaxheight", peermaxheight, "tipheight", tipheight)

				chain.FetchBlockHeaders(1, peermaxheight, pid)
			}
		}
	}
}

//本节点是否已经追赶上主链高度，追赶上之后通知本节点的共识模块开始挖矿
func (chain *BlockChain) IsCaughtUp() bool {

	height := chain.GetBlockHeight()

	peerMaxBlklock.Lock()
	defer peerMaxBlklock.Unlock()

	// peer中只有自己节点，没有其他节点
	if chain.peerList == nil {
		synlog.Debug("IsCaughtUp has no peers")
		return chain.cfg.SingleMode
	}

	var maxPeerHeight int64 = -1
	peersNo := 0
	for _, peer := range chain.peerList {
		if peer != nil && maxPeerHeight < peer.Height {
			maxPeerHeight = peer.Height
		}
		peersNo++
	}

	isCaughtUp := (height > 0 || time.Now().Sub(chain.startTime) > 60*time.Second) && (maxPeerHeight == 0 || height >= maxPeerHeight)

	synlog.Debug("IsCaughtUp", "IsCaughtUp ", isCaughtUp, "height", height, "maxPeerHeight", maxPeerHeight, "peersNo", peersNo)
	return isCaughtUp
}

//获取ntp时间是否同步状态
func GetNtpClockSyncStatus() bool {
	ntpClockSynclock.Lock()
	defer ntpClockSynclock.Unlock()
	return isNtpClockSync
}

//定时更新ntp时间同步状态
func UpdateNtpClockSyncStatus(Sync bool) {
	ntpClockSynclock.Lock()
	defer ntpClockSynclock.Unlock()
	isNtpClockSync = Sync
}
