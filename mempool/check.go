package mempool

import (
	"errors"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *Mempool) CheckTx(msg queue.Message) queue.Message {
	// 判断消息是否含有nil交易
	if msg.GetData() == nil {
		msg.Data = types.ErrEmptyTx
		return msg
	}
	// 检查交易是否为重复交易
	tx := msg.GetData().(*types.Transaction)
	if mem.addedTxs.Contains(string(tx.Hash())) {
		msg.Data = types.ErrDupTx
		return msg
	}
	mem.addedTxs.Add(string(tx.Hash()), nil)
	// 检查交易费是否小于最低值
	err := tx.Check(mem.minFee)
	if err != nil {
		msg.Data = err
		return msg
	}
	// 检查交易账户在Mempool中是否存在过多交易
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if mem.TxNumOfAccount(from) >= maxTxNumPerAccount {
		msg.Data = types.ErrManyTx
		return msg
	}
	// 检查交易是否过期
	valid := mem.CheckExpireValid(msg)
	if !valid {
		msg.Data = types.ErrTxExpire
		return msg
	}
	return msg
}

// Mempool.CheckSignList检查交易签名是否合法
func (mem *Mempool) CheckSignList() {
	for i := 0; i < processNum; i++ {
		go func() {
			for data := range mem.signChan {
				ok := data.GetData().(*types.Transaction).CheckSign()
				if ok {
					// 签名正确，传入balanChan，待检查余额
					mem.balanChan <- data
				} else {
					mlog.Error("wrong tx", "err", types.ErrSign)
					data.Data = types.ErrSign
					mem.badChan <- data
				}
			}
		}()
	}
}

// Mempool.CheckTxList读取balanChan数据存入msgs，待检查交易账户余额
func (mem *Mempool) CheckTxList() {
	for {
		var msgs [1024]queue.Message
		n, err := readToChan(mem.balanChan, msgs[:], 1024)
		if err != nil {
			mlog.Error("CheckTxList.readToChan", "err", err)
			return
		}
		mem.checkTxList(msgs[0:n])
	}
}

// readToChan将ch中数据依次存入buf
func readToChan(ch chan queue.Message, buf []queue.Message, max int) (n int, err error) {
	i := 0
	//先读取一个如果没有数据，就会卡在这里
	data, ok := <-ch
	if !ok {
		return 0, errors.New("channel closed")
	}
	buf[i] = data
	i++
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return i, errors.New("channel closed")
			}
			buf[i] = data
			i++
			if i == max {
				return i, nil
			}
		default:
			return i, nil
		}
	}
}

// Mempool.checkTxList检查账户余额是否足够，并加入到Mempool，成功则传入goodChan，若加入Mempool失败则传入badChan
func (mem *Mempool) checkTxList(msgs []queue.Message) {
	txlist := &types.ExecTxList{}
	for i := range msgs {
		tx := msgs[i].GetData().(*types.Transaction)
		txlist.Txs = append(txlist.Txs, tx)
	}
	lastheader := mem.GetHeader()
	txlist.BlockTime = lastheader.BlockTime
	txlist.Height = lastheader.Height
	txlist.StateHash = lastheader.StateHash

	result, err := mem.checkTxListRemote(txlist)
	if err != nil {
		for i := range msgs {
			mlog.Error("wrong tx", "err", err)
			msgs[i].Data = err
			mem.badChan <- msgs[i]
		}
		return
	}
	for i := 0; i < len(result.Errs); i++ {
		err := result.Errs[i]
		if err == "" {
			err1 := mem.PushTx(msgs[i].GetData().(*types.Transaction))
			if err1 == nil {
				// 推入Mempool成功，传入goodChan，待回复消息
				mem.goodChan <- msgs[i]
			} else {
				mlog.Error("wrong tx", "err", err1)
				msgs[i].Data = err1
				mem.badChan <- msgs[i]
			}
		} else {
			mlog.Error("wrong tx", "err", err)
			msgs[i].Data = errors.New(err)
			mem.badChan <- msgs[i]
		}
	}
}
