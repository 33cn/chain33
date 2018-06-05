package mempool

import (
	"errors"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto/privacy"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
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
				ok := false
				tx := data.GetData().(*types.Transaction)
				sign := tx.GetSignature()
				if sign != nil {
					ok = tx.CheckSign()
					//if sign.Ty != types.RingBaseonED25519 {
					//	ok = tx.CheckSign()
					//} else {
					//	ok = mem.CheckRingSign(tx)
					//}
				}

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

func (mem *Mempool) CheckRingSign(tx *types.Transaction) bool {
	var action types.PrivacyAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return false
	}
	var ringSign types.RingSignature
	if err := types.Decode(tx.GetSignature().GetSignature(), &ringSign); err != nil {
		mlog.Error("CheckRingSign", "error=", err.Error())
		return false
	}
	if len(ringSign.GetItems()) <= 0 {
		mlog.Error("CheckRingSign", "ring signature is empty.")
		return false
	}
	var privacyInput *types.PrivacyInput
	var token string
	if action.Ty == types.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		privacyInput = action.GetPrivacy2Privacy().Input
		token = action.GetPrivacy2Privacy().Tokenname
	} else if action.Ty == types.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		privacyInput = action.GetPrivacy2Public().Input
		token = action.GetPrivacy2Public().Tokenname
	}

	ReqUTXOPubKeys := &types.ReqUTXOPubKeys{
		TokenName: token,
	}
	for _, keyInput := range privacyInput.Keyinput {
		groupUTXOGlobalIndex := &types.GroupUTXOGlobalIndex{
			Amount:          keyInput.Amount,
			UtxoGlobalIndex: keyInput.UtxoGlobalIndex,
		}
		ReqUTXOPubKeys.GroupUTXOGlobalIndex = append(ReqUTXOPubKeys.GroupUTXOGlobalIndex, groupUTXOGlobalIndex)
	}

	msg := mem.client.NewMessage("blockchain", types.EventGetUTXOPubKey, ReqUTXOPubKeys)
	err = mem.client.Send(msg, true)
	if err != nil {
		mlog.Error("CheckRingSign", "err", err.Error())
		return false
	}
	msg, err = mem.client.Wait(msg)
	if err != nil {
		return false
	}
	resUTXOPubKeys := msg.GetData().(*types.ResUTXOPubKeys)
	copytx := *tx
	copytx.Signature = nil
	data := types.Encode(&copytx)
	h := common.BytesToHash(data)
	for i, ringSignItem := range ringSign.GetItems() {
		if !privacy.CheckRingSignature(h.Bytes(), ringSignItem, resUTXOPubKeys.GroupUTXOPubKeys[i].Pubkey, privacyInput.Keyinput[i].KeyImage) {
			mlog.Error("CheckRingSignature", "Failed to CheckRingSignature for index", i)
			return false
		}
	}
	return true
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
