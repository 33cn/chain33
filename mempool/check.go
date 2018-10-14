package mempool

import (
	"errors"

	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *Mempool) checkTx(msg queue.Message) queue.Message {
	tx := msg.GetData().(types.TxGroup).Tx()
	// 检查接收地址是否合法
	if err := address.CheckAddress(tx.To); err != nil {
		msg.Data = types.ErrInvalidAddress
		return msg
	}
	// 检查交易是否为重复交易
	if mem.addedTxs.Contains(string(tx.Hash())) {
		msg.Data = types.ErrDupTx
		return msg
	}

	// 检查交易账户在Mempool中是否存在过多交易
	from := tx.From()
	if mem.TxNumOfAccount(from) >= maxTxNumPerAccount {
		msg.Data = types.ErrManyTx
		return msg
	}
	// 检查交易是否过期
	valid, err := mem.CheckExpireValid(msg)
	if !valid {
		msg.Data = err
		return msg
	}
	return msg
}

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *Mempool) CheckTxs(msg queue.Message) queue.Message {
	// 判断消息是否含有nil交易
	if msg.GetData() == nil {
		msg.Data = types.ErrEmptyTx
		return msg
	}
	header := mem.GetHeader()
	txmsg := msg.GetData().(*types.Transaction)
	//普通的交易
	tx := types.NewTransactionCache(txmsg)
	err := tx.Check(header.GetHeight(), mem.minFee)
	if err != nil {
		msg.Data = err
		return msg
	}
	//检查txgroup 中的每个交易
	txs, err := tx.GetTxGroup()
	if err != nil {
		msg.Data = err
		return msg
	}
	msg.Data = tx
	//普通交易
	if txs == nil {
		return mem.checkTx(msg)
	}
	//txgroup 的交易
	for i := 0; i < len(txs.Txs); i++ {
		msgitem := mem.checkTx(queue.Message{Data: txs.Txs[i]})
		if msgitem.Err() != nil {
			msg.Data = msgitem.Err()
			return msg
		}
	}
	return msg
}

// Mempool.checkTxList检查账户余额是否足够，并加入到Mempool，成功则传入goodChan，若加入Mempool失败则传入badChan
func (mem *Mempool) checkTxRemote(msg queue.Message) queue.Message {
	tx := msg.GetData().(types.TxGroup)
	txlist := &types.ExecTxList{}
	txlist.Txs = append(txlist.Txs, tx.Tx())

	lastheader := mem.GetHeader()
	txlist.BlockTime = lastheader.BlockTime
	txlist.Height = lastheader.Height
	txlist.StateHash = lastheader.StateHash
	// 增加这个属性，在执行器中会使用到
	txlist.Difficulty = uint64(lastheader.Difficulty)
	txlist.IsMempool = true
	result, err := mem.checkTxListRemote(txlist)
	if err != nil {
		msg.Data = err
		return msg
	}
	errstr := result.Errs[0]
	if errstr == "" {
		err1 := mem.PushTx(txlist.Txs[0])
		if err1 != nil {
			mlog.Error("wrong tx", "err", err1)
			msg.Data = err1
		}
		return msg
	}
	mlog.Error("wrong tx", "err", errstr)
	msg.Data = errors.New(errstr)
	return msg
}
