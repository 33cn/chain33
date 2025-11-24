package mempool

import (
	"strings"
	"sync/atomic"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/queue"
	nty "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
)

func (mem *Mempool) reply() {
	defer mlog.Info("piple line quit")
	defer mem.wg.Done()
	for m := range mem.out {
		if m.Err() != nil {
			m.Reply(mem.client.NewMessage("rpc", types.EventReply,
				&types.Reply{IsOk: false, Msg: []byte(m.Err().Error())}))
		} else {
			mem.sendTxToP2P(m.GetData().(types.TxGroup).Tx())
			m.Reply(mem.client.NewMessage("rpc", types.EventReply, &types.Reply{IsOk: true, Msg: nil}))
		}
	}
}

func (mem *Mempool) pipeLine() <-chan *queue.Message {

	//check sign
	step1 := func(data *queue.Message) *queue.Message {
		if data.Err() != nil {
			return data
		}
		return mem.checkSign(data)
	}
	chs := make([]<-chan *queue.Message, processNum)
	for i := 0; i < processNum; i++ {
		chs[i] = step(mem.done, mem.in, step1)
	}
	out1 := merge(mem.done, chs)

	//checktx remote
	step2 := func(data *queue.Message) *queue.Message {
		if data.Err() != nil {
			return data
		}
		return mem.checkTxRemote(data)
	}
	chs2 := make([]<-chan *queue.Message, processNum)
	for i := 0; i < processNum; i++ {
		chs2[i] = step(mem.done, out1, step2)
	}

	return merge(mem.done, chs2)
}

// 处理其他模块的消息
func (mem *Mempool) eventProcess() {
	defer mem.wg.Done()
	defer close(mem.in)
	//event process
	mem.out = mem.pipeLine()
	mlog.Info("mempool piple line start")
	mem.wg.Add(1)
	go mem.reply()

	for msg := range mem.client.Recv() {
		//NOTE: 由于部分msg做了内存回收处理，业务逻辑处理后不能对msg进行引用访问， 相关数据提前保存
		msgName := types.GetEventName(int(msg.Ty))
		mlog.Debug("mempool recv", "msgid", msg.ID, "msg", msgName)
		beg := types.Now()
		switch msg.Ty {
		case types.EventTx:
			mem.eventTx(msg)
		case types.EventGetMempool:
			// 消息类型EventGetMempool：获取mempool内所有交易
			mem.eventGetMempool(msg)
		case types.EventTxList:
			// 消息类型EventTxList：获取mempool中一定数量交易
			mem.eventTxList(msg)
		case types.EventDelTxList:
			// 消息类型EventDelTxList：获取mempool中一定数量交易，并把这些交易从mempool中删除
			mem.eventDelTxList(msg)
		case types.EventAddBlock:
			// 消息类型EventAddBlock：将添加到区块内的交易从mempool中删除
			mem.eventAddBlock(msg)
		case types.EventGetMempoolSize:
			// 消息类型EventGetMempoolSize：获取mempool大小
			mem.eventGetMempoolSize(msg)
		case types.EventGetLastMempool:
			// 消息类型EventGetLastMempool：获取最新十条加入到mempool的交易
			mem.eventGetLastMempool(msg)
		case types.EventDelBlock:
			// 回滚区块，把该区块内交易重新加回mempool
			mem.eventDelBlock(msg)
		case types.EventGetAddrTxs:
			// 获取mempool中对应账户（组）所有交易
			mem.eventGetAddrTxs(msg)
		case types.EventGetProperFee:
			// 获取对应排队策略中合适的手续费
			mem.eventGetProperFee(msg)
			// 消息类型EventTxListByHash：通过hash获取对应的tx列表
		case types.EventTxListByHash:
			mem.eventTxListByHash(msg)
		case types.EventCheckTxsExist:
			mem.eventCheckTxsExist(msg)
		case types.EventAddDelayTx:
			mem.eventAddDelayTx(msg)

		default:
		}
		mlog.Debug("mempool", "cost", types.Since(beg), "msg", msgName)
	}
}

// EventTx 初步筛选后存入mempool
func (mem *Mempool) eventTx(msg *queue.Message) {
	if !mem.getSync() {
		msg.Reply(mem.client.NewMessage("", types.EventReply, &types.Reply{Msg: []byte(types.ErrNotSync.Error())}))
		mlog.Debug("wrong tx", "err", types.ErrNotSync.Error())
	} else {
		checkedMsg := mem.checkTxs(msg)
		select {
		case mem.in <- checkedMsg:
		case <-mem.done:
		}
	}
}

// EventGetMempool 获取Mempool内所有交易
func (mem *Mempool) eventGetMempool(msg *queue.Message) {
	var isAll bool
	if msg.GetData() == nil {
		isAll = false
	} else {
		isAll = msg.GetData().(*types.ReqGetMempool).GetIsAll()
	}
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
		&types.ReplyTxList{Txs: mem.filterTxList(0, nil, isAll)}))
}

// EventDelTxList 获取Mempool中一定数量交易，并把这些交易从Mempool中删除
func (mem *Mempool) eventDelTxList(msg *queue.Message) {
	hashList := msg.GetData().(*types.TxHashList)
	if len(hashList.GetHashes()) == 0 {
		msg.ReplyErr("EventDelTxList", types.ErrSize)
	} else {
		err := mem.RemoveTxs(hashList)
		msg.ReplyErr("EventDelTxList", err)
	}
}

// EventTxList 获取mempool中一定数量交易
func (mem *Mempool) eventTxList(msg *queue.Message) {
	hashList := msg.GetData().(*types.TxHashList)
	if hashList.Count <= 0 {
		msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, types.ErrSize))
		mlog.Error("not an valid size", "msg", msg)
	} else {
		txList := mem.getTxList(hashList)
		msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, &types.ReplyTxList{Txs: txList}))
	}
}

// EventAddBlock 将添加到区块内的交易从mempool中删除
func (mem *Mempool) eventAddBlock(msg *queue.Message) {
	block := msg.GetData().(*types.BlockDetail).Block
	height := mem.Height()
	lastHeader := mem.GetHeader()
	if block.Height > height || (block.Height == 0 && height == 0) {
		header := &types.Header{}
		header.BlockTime = block.BlockTime
		header.Height = block.Height
		header.StateHash = block.StateHash
		mem.setHeader(header)
	}
	//同步状态等mempool中不存在交易时，不需要执行操作
	if mem.Size() > 0 {
		mem.RemoveTxsOfBlock(block)
		mem.removeExpired()
	}
	// 检测是否存在延时存证交易，并将其中的延时交易进行暂存
	mem.addDelayTx(mem.cache.delayCache, block)
	// 区块高度增长，推送延时到期的延时交易
	mem.pushExpiredDelayTx(mem.cache.delayCache, lastHeader.GetBlockTime(),
		block.GetBlockTime(), block.GetHeight())

}

// add delay tx from new block
func (mem *Mempool) addDelayTx(cache *delayTxCache, block *types.Block) {

	// resolve commit delay tx type
	for _, tx := range block.GetTxs() {

		if !strings.Contains(string(tx.Execer), nty.NoneX) {
			continue
		}

		action := &nty.NoneAction{}
		if err := types.Decode(tx.Payload, action); err != nil ||
			action.Ty != nty.TyCommitDelayTxAction ||
			len(action.GetCommitDelayTx().GetDelayTx()) <= 0 {
			continue
		}
		commitInfo := action.GetCommitDelayTx()
		tx := &types.Transaction{}
		txByte, err := common.FromHex(commitInfo.GetDelayTx())
		if err != nil || types.Decode(txByte, tx) != nil {
			mlog.Error("addDelayTx", "txHash", common.ToHex(tx.Hash()),
				"decode delay tx err", err)
			continue
		}

		delayTx := &types.DelayTx{}
		delayTx.Tx = tx
		delayTx.EndDelayTime = commitInfo.GetRelativeDelayTime() + block.GetBlockTime()
		if commitInfo.GetRelativeDelayTime() <= 0 {
			delayTx.EndDelayTime = commitInfo.GetRelativeDelayHeight() + block.GetHeight()
		}
		if err := cache.addDelayTx(delayTx); err != nil {
			mlog.Error("addDelayTx", "txHash", common.ToHex(tx.Hash()),
				"delayTxHash", common.ToHex(delayTx.Tx.Hash()), "add delay tx cache error", err)
		}
	}
}

// push expired delay tx to mempool
func (mem *Mempool) pushExpiredDelayTx(delayCache *delayTxCache, lastBlockTime,
	currBlockTime, currBlockHeight int64) {

	delayTxList := delayCache.delExpiredTxs(
		lastBlockTime, currBlockTime, currBlockHeight)
	if len(delayTxList) == 0 {
		return
	}

	// 阻塞时异步发送，避免mempool消息处理死锁, 延时交易属于低频操作，阻塞时协程开销不会很大
	select {
	case mem.delayTxListChan <- delayTxList:
	default:
		go func() { mem.delayTxListChan <- delayTxList }()
	}
}

// EventGetMempoolSize 获取mempool大小
func (mem *Mempool) eventGetMempoolSize(msg *queue.Message) {
	memSize := int64(mem.Size())
	msg.Reply(mem.client.NewMessage("rpc", types.EventMempoolSize,
		&types.MempoolSize{Size: memSize}))
}

// EventGetLastMempool 获取最新十条加入到mempool的交易
func (mem *Mempool) eventGetLastMempool(msg *queue.Message) {
	txList := mem.GetLatestTx()
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
		&types.ReplyTxList{Txs: txList}))
}

// EventDelBlock 回滚区块，把该区块内交易重新加回mempool
func (mem *Mempool) eventDelBlock(msg *queue.Message) {
	block := msg.GetData().(*types.BlockDetail).Block
	if block.Height != mem.GetHeader().GetHeight() {
		return
	}
	lastHeader, err := mem.GetLastHeader()
	if err != nil {
		mlog.Error(err.Error())
		return
	}
	h := lastHeader.(*queue.Message).Data.(*types.Header)
	mem.setHeader(h)
	mem.delBlock(block)
}

// eventGetAddrTxs 获取mempool中对应账户（组）所有交易
func (mem *Mempool) eventGetAddrTxs(msg *queue.Message) {
	addrs := msg.GetData().(*types.ReqAddrs)
	txlist := mem.GetAccTxs(addrs)
	msg.Reply(mem.client.NewMessage("", types.EventReplyAddrTxs, txlist))
}

// eventGetProperFee 获取排队策略中合适的手续费率
func (mem *Mempool) eventGetProperFee(msg *queue.Message) {
	req, _ := msg.GetData().(*types.ReqProperFee)
	properFee := mem.GetProperFeeRate(req)
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyProperFee,
		&types.ReplyProperFee{ProperFee: properFee}))
}

func (mem *Mempool) checkSign(data *queue.Message) *queue.Message {
	tx, ok := data.GetData().(types.TxGroup)
	if ok && tx.CheckSign(atomic.LoadInt64(&mem.currHeight)+1) {
		return data
	}
	mlog.Error("wrong tx", "err", types.ErrSign)
	data.Data = types.ErrSign
	return data
}

// eventTxListByHash 通过hash获取tx列表
func (mem *Mempool) eventTxListByHash(msg *queue.Message) {
	shashList := msg.GetData().(*types.ReqTxHashList)
	replytxList := mem.getTxListByHash(shashList)
	msg.Reply(mem.client.NewMessage("", types.EventReplyTxList, replytxList))
}

// eventCheckTxsExist 查找交易是否存在
func (mem *Mempool) eventCheckTxsExist(msg *queue.Message) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	reply := &types.ReplyCheckTxsExist{}
	//非同步状态时mempool通常不存在交易，直接返回不做比对
	if mem.sync {
		hashes := msg.GetData().(*types.ReqCheckTxsExist).TxHashes
		reply.ExistFlags = make([]bool, len(hashes))
		for i, hash := range hashes {
			reply.ExistFlags[i] = mem.cache.Exist(types.Bytes2Str(hash))
			if reply.ExistFlags[i] {
				reply.ExistCount++
			}
		}
	}
	msg.Reply(mem.client.NewMessage("", types.EventReply, reply))
}

// 添加延时交易
func (mem *Mempool) eventAddDelayTx(msg *queue.Message) {

	err := types.ErrInvalidParam
	if delayTx, ok := msg.GetData().(*types.DelayTx); ok {
		err = mem.cache.delayCache.addDelayTx(delayTx)
	}
	replyMsg := mem.client.NewMessage("rpc", types.EventReply, nil)
	if err != nil {
		replyMsg.Data = &types.Reply{Msg: []byte(err.Error())}
	} else {
		replyMsg.Data = &types.Reply{IsOk: true}
	}
	msg.Reply(replyMsg)
}
