package mempool

import (
	"github.com/33cn/chain33/queue"
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
		mlog.Debug("mempool recv", "msgid", msg.ID, "msg", types.GetEventName(int(msg.Ty)))
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
		default:
		}
		mlog.Debug("mempool", "cost", types.Since(beg), "msg", types.GetEventName(int(msg.Ty)))
	}
}

//EventTx 初步筛选后存入mempool
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
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyTxList,
		&types.ReplyTxList{Txs: mem.filterTxList(0, nil)}))
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
	if block.Height > mem.Height() || (block.Height == 0 && mem.Height() == 0) {
		header := &types.Header{}
		header.BlockTime = block.BlockTime
		header.Height = block.Height
		header.StateHash = block.StateHash
		mem.setHeader(header)
	}
	mem.RemoveTxsOfBlock(block)
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

// eventGetProperFee 获取排队策略中合适的手续费
func (mem *Mempool) eventGetProperFee(msg *queue.Message) {
	properFee := mem.cache.qcache.GetProperFee()
	msg.Reply(mem.client.NewMessage("rpc", types.EventReplyProperFee,
		&types.ReplyProperFee{ProperFee: properFee}))
}

func (mem *Mempool) checkSign(data *queue.Message) *queue.Message {
	tx, ok := data.GetData().(types.TxGroup)
	if ok && tx.CheckSign() {
		return data
	}
	mlog.Error("wrong tx", "err", types.ErrSign)
	data.Data = types.ErrSign
	return data
}
