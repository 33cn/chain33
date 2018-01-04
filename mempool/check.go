package mempool

import (
	"errors"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *Mempool) CheckTx(msg queue.Message) queue.Message {
	if msg.GetData() == nil { // 判断消息是否含有nil交易 d
		msg.Data = e09
		return msg
	}
	tx := msg.GetData().(*types.Transaction)
	// 检查交易消息是否过大
	if mem.addedTxs.Contains(string(tx.Hash())) {
		msg.Data = e10
		return msg
	}
	mem.addedTxs.Add(string(tx.Hash()), nil)
	if len(types.Encode(tx)) > int(maxMsgByte) {
		msg.Data = e06
		return msg
	}
	// 检查交易费是否小于最低值
	if tx.Fee < mem.GetMinFee() {
		msg.Data = e02
		return msg
	}
	// 检查交易账户在Mempool中是否存在过多交易
	from := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if mem.TxNumOfAccount(from) >= maxTxNumPerAccount {
		msg.Data = e03
		return msg
	}
	valid := mem.CheckExpireValid(msg) // 检查交易是否过期
	if !valid {
		msg.Data = e07
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
					mlog.Info("wrong tx", "err", e04)
					data.Data = e04
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
	var msgTmp []queue.Message
	var accTmp []*types.Account
	for index, ad := range addrs {
		if ac, exists := mem.accountCache[ad]; exists {
			msgTmp = append(msgTmp, msgs[index])
			accTmp = append(accTmp, ac)
			msgs = append(msgs[:index], msgs[index+1:]...)
			addrs = append(addrs[:index], addrs[index+1:]...)
		}
	}
	accs, err := account.LoadAccounts(mem.memQueue, addrs)

	if err != nil {
		mlog.Error("loadaccounts", "err", err)
		for m := range msgs {
			mlog.Info("wrong tx", "err", e08)
			msgs[m].Data = e08
			mem.badChan <- msgs[m]
		}

		return
	}

	for i := range accs {
		mem.accountCache[addrs[i]] = accs[i]
	}

	msgs = append(msgs, msgTmp...)
	accs = append(accs, accTmp...)

	for i := range msgs {
		tx := msgs[i].GetData().(*types.Transaction)

		if accs[i].GetBalance() >= 10*tx.Fee {
			// 交易账户余额充足，推入Mempool
			err := mem.PushTx(tx)
			if err == nil {
				mem.goodChan <- msgs[i]
			} else {
				mlog.Info("wrong tx", "err", err)
				msgs[i].Data = err
				mem.badChan <- msgs[i]
			}
		} else {
			mlog.Info("wrong tx", "err", e05)
			msgs[i].Data = e05
			mem.badChan <- msgs[i]
		}
	}
}

// Mempool.CheckExpireValid检查交易过期有效性，过期返回false，未过期返回true
func (mem *Mempool) CheckExpireValid(msg queue.Message) bool {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	tx := msg.GetData().(*types.Transaction)
	if tx.IsExpire(mem.height, mem.blockTime) {
		return false
	}
	return true
}
