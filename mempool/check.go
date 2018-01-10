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
		msg.Data = emptyTxErr
		return msg
	}
	// 检查交易是否为重复交易
	tx := msg.GetData().(*types.Transaction)
	if mem.addedTxs.Contains(string(tx.Hash())) {
		msg.Data = dupTxErr
		return msg
	}
	mem.addedTxs.Add(string(tx.Hash()), nil)
	// 检查交易消息是否过大
	txSize := types.Size(tx)
	if txSize > int(maxMsgByte) {
		msg.Data = bigMsgErr
		return msg
	}
	// 检查交易费是否小于最低值
	var realFee int64
	if txSize/1000.0-txSize/1000 == 0 {
		realFee = int64(txSize/1000) * mem.GetMinFee()
	} else {
		realFee = int64(txSize/1000+1) * mem.GetMinFee()
	}
	if tx.Fee < realFee {
		msg.Data = lowFeeErr
		return msg
	}
	// 检查交易账户在Mempool中是否存在过多交易
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if mem.TxNumOfAccount(from) >= maxTxNumPerAccount {
		msg.Data = manyTxErr
		return msg
	}
	// 检查交易是否过期
	valid := mem.CheckExpireValid(msg)
	if !valid {
		msg.Data = expireErr
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
					mlog.Info("wrong tx", "err", signErr)
					data.Data = signErr
					mem.badChan <- data
				}
			}
		}()
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
	return i, nil
}

// Mempool.CheckBalanList读取balanChan数据存入msgs，待检查交易账户余额
func (mem *Mempool) CheckBalanList() {
	for {
		var msgs [1024]queue.Message
		var addrs [1024]string
		n, err := readToChan(mem.balanChan, msgs[:], 1024)

		if err != nil {
			mlog.Error("CheckBalanList.readToChan", "err", err)
			return
		}

		for i := 0; i < n; i++ {
			data := msgs[i]
			pubKey := data.GetData().(*types.Transaction).GetSignature().GetPubkey()
			addrs[i] = account.PubKeyToAddress(pubKey).String()
		}

		mem.checkBalance(msgs[0:n], addrs[0:n])
	}
}

// Mempool.checkBalance检查交易账户余额
func (mem *Mempool) checkBalance(msgs []queue.Message, addrs []string) {
	if len(msgs) != len(addrs) {
		mlog.Error("msgs size not equals addrs size")
		return
	}
	accs, err := account.LoadAccountsDB(mem.GetDB(), addrs)
	if err != nil {
		mlog.Error("loadaccounts", "err", err)
		for m := range msgs {
			mlog.Info("wrong tx", "err", loadAccountsErr)
			msgs[m].Data = loadAccountsErr
			mem.badChan <- msgs[m]
		}
		return
	}
	for i := range msgs {
		tx := msgs[i].GetData().(*types.Transaction)
		if accs[i].GetBalance() >= 10*tx.Fee {
			// 交易账户余额充足，推入Mempool
			err := mem.PushTx(tx)
			if err == nil {
				// 推入Mempool成功，传入goodChan，待回复消息
				mem.goodChan <- msgs[i]
			} else {
				mlog.Info("wrong tx", "err", err)
				msgs[i].Data = err
				mem.badChan <- msgs[i]
			}
		} else {
			mlog.Info("wrong tx", "err", lowBalanceErr)
			msgs[i].Data = lowBalanceErr
			mem.badChan <- msgs[i]
		}
	}
}
