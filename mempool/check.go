package mempool

import (
	"errors"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *Mempool) checkTx(msg queue.Message) queue.Message {
	tx := msg.GetData().(types.TxGroup).Tx()
	// 过滤掉挖矿交易
	if "ticket" == string(tx.Execer) {
		var action types.TicketAction
		err := types.Decode(tx.Payload, &action)
		if err != nil {
			msg.Data = err
			return msg
		}
		if action.Ty == types.TicketActionMiner && action.GetMiner() != nil {
			msg.Data = types.ErrMinerTx
			return msg
		}
	}
	// 检查接收地址是否合法
	if err := account.CheckAddress(tx.To); err != nil {
		msg.Data = types.ErrInvalidAddress
		return msg
	}
	// 检查交易是否为重复交易
	if mem.addedTxs.Contains(string(tx.Hash())) {
		msg.Data = types.ErrDupTx
		return msg
	}
	mem.addedTxs.Add(string(tx.Hash()), nil)

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

func (mem *Mempool) CheckSignFromAuth(tx *types.Transaction) bool {
	if mem.client == nil {
		panic("client not bind message queue.")
	}

	req := &types.ReqAuthSignCheck{}
	req.Tx = tx
	msg := mem.client.NewMessage("authority", types.EventAuthorityCheckTx, req)
	err := mem.client.Send(msg, true)
	if err != nil {
		mlog.Error("check signature from authority failed", "err", err.Error())
		return false
	}
	msg, err = mem.client.Wait(msg)
	if err != nil {
		mlog.Error("check signature from authority failed", "err", err.Error())
		return false
	}

	return msg.GetData().(*types.RespAuthSignCheck).Result
}

// Mempool.CheckTxList初步检查并筛选交易消息
func (mem *Mempool) CheckTxs(msg queue.Message) queue.Message {
	// 判断消息是否含有nil交易
	if msg.GetData() == nil {
		msg.Data = types.ErrEmptyTx
		return msg
	}
	txmsg := msg.GetData().(*types.Transaction)
	//普通的交易
	tx := types.NewTransactionCache(txmsg)
	err := tx.Check(mem.minFee)
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

// Mempool.CheckSignList检查交易签名是否合法
func (mem *Mempool) CheckSignList() {
	var result bool = false
	for i := 0; i < processNum; i++ {
		go func() {
			for data := range mem.signChan {
				tx, ok := data.GetData().(types.TxGroup)
				if tx.Tx().GetSignature().Ty == types.SIG_TYPE_AUTHORITY {
					result = mem.CheckSignFromAuth(tx.Tx())
					if result {
						// 签名正确，联盟链跳过余额检查
						mem.PushTx(tx.Tx())
						mem.goodChan <- data
					} else {
						mlog.Error("wrong tx", "err", types.ErrSign)
						data.Data = types.ErrSign
						mem.badChan <- data
					}
				} else {
					if ok && tx.CheckSign() {
						mem.balanChan <- data
					} else {
						mlog.Error("wrong tx", "err", types.ErrSign)
						data.Data = types.ErrSign
						mem.badChan <- data
					}
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
		tx := msgs[i].GetData().(types.TxGroup)
		txlist.Txs = append(txlist.Txs, tx.Tx())
	}
	lastheader := mem.GetHeader()
	txlist.BlockTime = lastheader.BlockTime
	txlist.Height = lastheader.Height
	txlist.StateHash = lastheader.StateHash
	// 增加这个属性，在执行器中会使用到
	txlist.Difficulty = uint64(lastheader.Difficulty)

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
			err1 := mem.PushTx(txlist.Txs[i])
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
