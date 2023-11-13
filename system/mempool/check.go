package mempool

import (
	"bytes"
	"errors"
	"sort"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/common"

	"github.com/33cn/chain33/util"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// CheckExpireValid 检查交易过期有效性，过期返回false，未过期返回true
func (mem *Mempool) CheckExpireValid(msg *queue.Message) (bool, error) {
	mem.proxyMtx.Lock()
	defer mem.proxyMtx.Unlock()
	if mem.header == nil {
		return false, types.ErrHeaderNotSet
	}
	tx := msg.GetData().(types.TxGroup).Tx()
	ok := mem.checkExpireValid(tx)
	if !ok {
		return ok, types.ErrTxExpire
	}
	return ok, nil
}

// checkTxListRemote 发送消息给执行模块检查交易
func (mem *Mempool) checkTxListRemote(txlist *types.ExecTxList) (*types.ReceiptCheckTxList, error) {
	if mem.client == nil {
		panic("client not bind message queue.")
	}
	msg := mem.client.NewMessage("execs", types.EventCheckTx, txlist)
	err := mem.client.Send(msg, true)
	if err != nil {
		mlog.Error("execs closed", "err", err.Error())
		return nil, err
	}
	reply, err := mem.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	txList := reply.GetData().(*types.ReceiptCheckTxList)
	mem.client.FreeMessage(msg, reply)
	return txList, nil
}

func (mem *Mempool) checkExpireValid(tx *types.Transaction) bool {
	types.AssertConfig(mem.client)
	//进入mempool中的交易可能被下一个区块打包，这里用下一个区块的高度和时间作为交易过期判定
	if tx.IsExpire(mem.client.GetConfig(), mem.header.GetHeight()+1, mem.header.GetBlockTime()) {
		return false
	}
	if tx.Expire > 1000000000 && tx.Expire < types.Now().Unix()+int64(time.Minute/time.Second) {
		return false
	}
	return true
}

// CheckTx 初步检查并筛选交易消息
func (mem *Mempool) checkTx(msg *queue.Message) *queue.Message {
	tx := msg.GetData().(types.TxGroup).Tx()
	// 检查接收地址是否合法
	if err := address.CheckAddress(tx.To, atomic.LoadInt64(&mem.currHeight)); err != nil {
		msg.Data = types.ErrInvalidAddress
		return msg
	}
	// 检查交易账户在mempool中是否存在过多交易
	from := tx.From()
	if mem.TxNumOfAccount(from) >= mem.cfg.MaxTxNumPerAccount {
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

// CheckTxs 初步检查并筛选交易消息
func (mem *Mempool) checkTxs(msg *queue.Message) *queue.Message {
	// 判断消息是否含有nil交易
	if msg.GetData() == nil {
		msg.Data = types.ErrEmptyTx
		return msg
	}

	cacheTx := types.NewTransactionCache(msg.GetData().(*types.Transaction))
	msg.Data = cacheTx

	cfg := mem.client.GetConfig()
	// 转发的交易由主链验证, 平行链忽略基础检查
	if types.IsForward2MainChainTx(cfg, cacheTx.Transaction) {
		return msg
	}

	err := cacheTx.Check(cfg, mem.GetHeader().GetHeight()+1, mem.cfg.MinTxFeeRate, mem.cfg.MaxTxFee)
	if err == nil && mem.cfg.IsLevelFee {
		err = mem.checkLevelFee(cacheTx)
	}

	var txs *types.Transactions
	if err == nil {
		txs, err = cacheTx.GetTxGroup()
	}

	if err != nil {
		msg.Data = err
		return msg
	}

	//普通交易
	if txs == nil {
		return mem.checkTx(msg)
	}
	//txgroup 的交易
	for i := 0; i < len(txs.Txs); i++ {
		msgitem := mem.checkTx(&queue.Message{Data: txs.Txs[i]})
		if msgitem.Err() != nil {
			msg.Data = msgitem.Err()
			return msg
		}
	}
	return msg
}

// checkLevelFee 检查阶梯手续费
func (mem *Mempool) checkLevelFee(tx *types.TransactionCache) error {
	//获取mempool里所有交易手续费总和
	feeRate := mem.getLevelFeeRate(mem.cfg.MinTxFeeRate, 0, 0)
	totalfee, err := tx.GetTotalFee(feeRate)
	if err != nil {
		return err
	}
	if tx.Fee < totalfee {
		return types.ErrTxFeeTooLow
	}
	return nil
}

// checkTxRemote 检查账户余额是否足够，并加入到Mempool，成功则传入goodChan，若加入Mempool失败则传入badChan
func (mem *Mempool) checkTxRemote(msg *queue.Message) *queue.Message {
	tx := msg.GetData().(types.TxGroup)
	lastheader := mem.GetHeader()

	//add check dup tx需要区分单笔交易/交易组
	temtxlist := &types.ExecTxList{}
	txGroup, err := tx.GetTxGroup()
	if err != nil {
		msg.Data = err
		return msg
	}
	if txGroup == nil {
		temtxlist.Txs = append(temtxlist.Txs, tx.Tx())
	} else {
		temtxlist.Txs = append(temtxlist.Txs, txGroup.GetTxs()...)
	}

	temtxlist.Height = lastheader.Height
	newtxs, err := util.CheckDupTx(mem.client, temtxlist.Txs, temtxlist.Height)
	if err != nil {
		msg.Data = err
		return msg
	}
	if len(newtxs) != len(temtxlist.Txs) {
		msg.Data = types.ErrDupTx
		return msg
	}

	//exec模块检查效率影响系统性能， 支持关闭
	if !mem.cfg.DisableExecCheck {
		txlist := &types.ExecTxList{}
		txlist.Txs = append(txlist.Txs, tx.Tx())
		txlist.BlockTime = lastheader.BlockTime
		txlist.Height = lastheader.Height
		txlist.StateHash = lastheader.StateHash
		// 增加这个属性，在执行器中会使用到
		txlist.Difficulty = uint64(lastheader.Difficulty)
		txlist.IsMempool = true

		result, err := mem.checkTxListRemote(txlist)

		if err == nil && result.Errs[0] != "" {
			err = errors.New(result.Errs[0])
		}
		if err != nil {
			mlog.Error("checkTxRemote", "txHash", common.ToHex(tx.Tx().Hash()), "checkTxListRemoteErr", err)
			msg.Data = err
			return msg
		}
	}

	//检查mempool内是否有相同的tx.nonce
	err = mem.evmTxNonceCheck(tx.Tx())
	if err != nil {
		msg.Data = err
		return msg
	}

	err = mem.PushTx(tx.Tx())
	if err != nil {
		if err == types.ErrMemFull {
			//has a sleep
			time.Sleep(time.Millisecond * 200)
		}
		mlog.Error("checkTxRemote", "push err", err)
		msg.Data = err
	}
	return msg
}

// evmTxNonceCheck 检查eth noce 是否有重复值，如果有的话，需要比较txFee大小，用于替换较小的fee的那笔交易
func (mem *Mempool) evmTxNonceCheck(tx *types.Transaction) error {
	if !types.IsEthSignID(tx.Tx().GetSignature().GetTy()) {
		return nil
	}

	//较小的nonce 则返回错误，不被允许进入mempool
	if tx.GetNonce() < mem.getCurrentNonce(tx.From()) {
		return types.ErrLowNonce
	}
	details := mem.GetAccTxs(&types.ReqAddrs{Addrs: []string{tx.From()}})
	txs := details.GetTxs()
	txs = append(txs, &types.TransactionDetail{Tx: tx, Index: int64(len(txs))})
	if len(txs) > 1 {
		sort.SliceStable(txs, func(i, j int) bool { //nonce asc
			return txs[i].Tx.GetNonce() < txs[j].Tx.GetNonce()
		})
		//遇到相同的Nonce ,较低的手续费的交易将被删除
		for i, stx := range txs {
			if bytes.Equal(stx.Tx.Hash(), tx.Hash()) {
				continue
			}
			if txs[i].GetTx().GetNonce() == tx.GetNonce() {
				mlog.Info("evmTxNonceCheck", "from:", tx.From(), "pre tx hash", common.ToHex(txs[i].GetTx().Hash()), "detect transaction acceleration action:", "reject")
				return errors.New("disable transaction acceleration")
			}
		}

	}

	return nil

}
