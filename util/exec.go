package util

import (
	"errors"

	"gitlab.33.cn/chain33/chain33/common"
	log "gitlab.33.cn/chain33/chain33/common/log/log15"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

func CheckBlock(client queue.Client, block *types.BlockDetail) error {
	req := block
	msg := client.NewMessage("consensus", types.EventCheckBlock, req)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		return err
	}
	reply := resp.GetData().(*types.Reply)
	if reply.IsOk {
		return nil
	}
	return errors.New(string(reply.GetMsg()))
}

func ExecTx(client queue.Client, prevStateRoot []byte, block *types.Block) *types.Receipts {
	list := &types.ExecTxList{prevStateRoot, block.Txs, block.BlockTime, block.Height, uint64(block.Difficulty), false}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	receipts := resp.GetData().(*types.Receipts)
	return receipts
}

func ExecKVMemSet(client queue.Client, prevStateRoot []byte, height int64, kvset []*types.KeyValue, sync bool) []byte {
	set := &types.StoreSet{prevStateRoot, kvset, height}
	setwithsync := &types.StoreSetWithSync{set, sync}

	msg := client.NewMessage("store", types.EventStoreMemSet, setwithsync)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	hash := resp.GetData().(*types.ReplyHash)
	return hash.GetHash()
}

func ExecKVSetCommit(client queue.Client, hash []byte) error {
	req := &types.ReqHash{hash}
	msg := client.NewMessage("store", types.EventStoreCommit, req)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	return nil
}

func ExecKVSetRollback(client queue.Client, hash []byte) error {
	req := &types.ReqHash{hash}
	msg := client.NewMessage("store", types.EventStoreRollback, req)
	client.Send(msg, true)
	msg, err := client.Wait(msg)
	if err != nil {
		return err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	return nil
}

func CheckTxDupInner(txs []*types.TransactionCache) (ret []*types.TransactionCache) {
	dupMap := make(map[string]bool)
	for _, tx := range txs {
		hash := string(tx.Hash())
		if _, ok := dupMap[hash]; ok {
			continue
		}
		dupMap[hash] = true
		ret = append(ret, tx)
	}
	return ret
}

func CheckTxDup(client queue.Client, txs []*types.TransactionCache, height int64) (transactions []*types.TransactionCache, err error) {
	var checkHashList types.TxHashList
	if types.IsFork(height, "ForkCheckTxDup") {
		txs = CheckTxDupInner(txs)
	}
	for _, tx := range txs {
		checkHashList.Hashes = append(checkHashList.Hashes, tx.Hash())
		checkHashList.Expire = append(checkHashList.Expire, tx.GetExpire())
	}
	checkHashList.Count = height
	hashList := client.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
	client.Send(hashList, true)
	dupTxList, err := client.Wait(hashList)
	if err != nil {
		return nil, err
	}
	dupTxs := dupTxList.GetData().(*types.TxHashList).Hashes
	dupMap := make(map[string]bool)
	for _, hash := range dupTxs {
		dupMap[string(hash)] = true
		log.Debug("CheckTxDup", "TxDuphash", common.ToHex(hash))
	}
	for _, tx := range txs {
		hash := tx.Hash()
		if dupMap[string(hash)] {
			continue
		}
		transactions = append(transactions, tx)
	}
	return transactions, nil
}

//上报指定错误信息到指定模块，目前只支持从store，blockchain，wallet写数据库失败时上报错误信息到wallet模块，
//然后由钱包模块前端定时调用显示给客户
func ReportErrEventToFront(logger log.Logger, client queue.Client, frommodule string, tomodule string, err error) {
	if client == nil || len(tomodule) == 0 || len(frommodule) == 0 || err == nil {
		logger.Error("SendErrEventToFront  input para err .")
		return
	}

	logger.Debug("SendErrEventToFront", "frommodule", frommodule, "tomodule", tomodule, "err", err)

	var reportErrEvent types.ReportErrEvent
	reportErrEvent.Frommodule = frommodule
	reportErrEvent.Tomodule = tomodule
	reportErrEvent.Error = err.Error()
	msg := client.NewMessage(tomodule, types.EventErrToFront, &reportErrEvent)
	client.Send(msg, false)
}
