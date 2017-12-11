package mempool

import (
	"errors"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *Mempool) CheckTxList() {
	for data := range mem.txChan {
		tx := data.GetData().(*types.Transaction)

		// 检查交易消息是否过大
		if len(types.Encode(tx)) > int(maxMsgByte) {
			data.Data = errors.New(e06)
			mem.badChan <- data
			break
		}

		// 检查交易费是否小于最低值
		if tx.Fee < mem.GetMinFee() {
			data.Data = errors.New(e02)
			mem.badChan <- data
			break
		}

		// 检查交易账户在Mempool中是否存在过多交易
		if mem.TxNumOfAccount(account.PubKeyToAddress(tx.GetSignature().GetPubkey())) >= 10 {
			data.Data = errors.New(e03)
			mem.badChan <- data
			break
		}

		// 传入signChan，待检查签名
		mem.signChan <- data
	}
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
					data.Data = errors.New(e04)
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
	accs, _ := account.LoadAccounts(mem.memQueue, addrs)
	for i := range msgs {
		tx := msgs[i].GetData().(*types.Transaction)

		if accs[i].Balance >= 10*tx.Fee {
			// 交易账户余额充足，推入Mempool
			ok, err := mem.PushTx(tx)
			if ok {
				// 推入Mempool成功，传入goodChan，待回复消息
				mem.goodChan <- msgs[i]
			} else {
				msgs[i].Data = errors.New(err)
				mem.badChan <- msgs[i]
			}
		} else {
			msgs[i].Data = errors.New(e05)
			mem.badChan <- msgs[i]
		}
	}
}

// Mempool.CheckExpire检查交易是否过期，未过期返回true，过期返回false
func (mem *Mempool) CheckExpire(msg queue.Message) bool {
	valid := msg.GetData().(*types.Transaction).Expire
	// Expire为0，返回true
	if valid == 0 {
		return true
	}

	if valid <= expireBound {
		// Expire小于1e9，为height
		if valid > (mem.Height()) { // 未过期
			return true
		} else { // 过期
			msg.Data = errors.New(e07)
			return false
		}
	} else {
		// Expire大于1e9，为blockTime
		if valid > (mem.BlockTime()) { // 未过期
			return true
		} else { // 过期
			msg.Data = errors.New(e07)
			return false
		}
	}
}
