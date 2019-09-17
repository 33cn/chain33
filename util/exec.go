// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"errors"

	"github.com/33cn/chain33/common"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

//CheckBlock : To check the block's validaty
func CheckBlock(client queue.Client, block *types.BlockDetail) error {
	req := block
	msg := client.NewMessage("consensus", types.EventCheckBlock, req)
	err := client.Send(msg, true)
	if err != nil {
		return err
	}
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

//ExecTx : To send lists of txs within a block to exector for execution
func ExecTx(client queue.Client, prevStateRoot []byte, block *types.Block) (*types.Receipts, error) {
	list := &types.ExecTxList{
		StateHash:  prevStateRoot,
		ParentHash: block.ParentHash,
		MainHash:   block.MainHash,
		MainHeight: block.MainHeight,
		Txs:        block.Txs,
		BlockTime:  block.BlockTime,
		Height:     block.Height,
		Difficulty: uint64(block.Difficulty),
		IsMempool:  false,
	}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	err := client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	receipts := resp.GetData().(*types.Receipts)
	return receipts, nil
}

//ExecKVMemSet : send kv values to memory store and set it in db
func ExecKVMemSet(client queue.Client, prevStateRoot []byte, height int64, kvset []*types.KeyValue, sync bool, upgrade bool) ([]byte, error) {
	set := &types.StoreSet{StateHash: prevStateRoot, KV: kvset, Height: height}
	setwithsync := &types.StoreSetWithSync{Storeset: set, Sync: sync, Upgrade: upgrade}

	msg := client.NewMessage("store", types.EventStoreMemSet, setwithsync)
	err := client.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.Wait(msg)
	if err != nil {
		return nil, err
	}
	hash := resp.GetData().(*types.ReplyHash)
	return hash.GetHash(), nil
}

//ExecKVSetCommit : commit the data set opetation to db
func ExecKVSetCommit(client queue.Client, hash []byte, upgrade bool) error {
	req := &types.ReqHash{Hash: hash, Upgrade: upgrade}
	msg := client.NewMessage("store", types.EventStoreCommit, req)
	err := client.Send(msg, true)
	if err != nil {
		return err
	}
	msg, err = client.Wait(msg)
	if err != nil {
		return err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	_ = hash
	return nil
}

//ExecKVSetRollback : do the db's roll back operation
func ExecKVSetRollback(client queue.Client, hash []byte) error {
	req := &types.ReqHash{Hash: hash}
	msg := client.NewMessage("store", types.EventStoreRollback, req)
	err := client.Send(msg, true)
	if err != nil {
		return err
	}
	msg, err = client.Wait(msg)
	if err != nil {
		return err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	_ = hash
	return nil
}

//DelDupTx 删除重复的交易
func DelDupTx(txs []*types.TransactionCache) (ret []*types.TransactionCache) {
	dupindex := make(map[string]int)
	hasdup := false
	for i, tx := range txs {
		if _, ok := dupindex[string(tx.Hash())]; ok {
			hasdup = true
		}
		dupindex[string(tx.Hash())] = i
	}
	//没有重复的情况下，不需要重新处理
	if !hasdup {
		return txs
	}
	index := 0
	for i, tx := range txs {
		lastindex := dupindex[string(tx.Hash())]
		if i == lastindex {
			txs[index] = tx
			index++
		}
	}
	return txs[0:index]
}

//CheckDupTx : check use txs []*types.Transaction and not []*types.TransactionCache
func CheckDupTx(client queue.Client, txs []*types.Transaction, height int64) (transactions []*types.Transaction, err error) {
	txcache := make([]*types.TransactionCache, len(txs))
	for i := 0; i < len(txcache); i++ {
		txcache[i] = &types.TransactionCache{Transaction: txs[i]}
	}
	cache, err := CheckTxDup(client, txcache, height)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(cache); i++ {
		transactions = append(transactions, cache[i].Transaction)
	}
	return transactions, nil
}

//CheckTxDup : check whether the tx is duplicated within the while chain
func CheckTxDup(client queue.Client, txs []*types.TransactionCache, height int64) (transactions []*types.TransactionCache, err error) {
	var checkHashList types.TxHashList
	if types.IsFork(height, "ForkCheckTxDup") {
		txs = DelDupTx(txs)
	}
	for _, tx := range txs {
		checkHashList.Hashes = append(checkHashList.Hashes, tx.Hash())
		checkHashList.Expire = append(checkHashList.Expire, tx.GetExpire())
	}
	checkHashList.Count = height
	hashList := client.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
	err = client.Send(hashList, true)
	if err != nil {
		log.Error("send", "to blockchain EventTxHashList msg err", err)
		return nil, err
	}
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

//ReportErrEventToFront 上报指定错误信息到指定模块，目前只支持从store，blockchain，wallet写数据库失败时上报错误信息到wallet模块，
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
	err = client.Send(msg, false)
	if err != nil {
		log.Error("send", "EventErrToFront msg err", err)
	}
}

//DelDupKey 删除重复的key
func DelDupKey(kvs []*types.KeyValue) []*types.KeyValue {
	dupindex := make(map[string]int)
	n := 0
	for _, kv := range kvs {
		skey := string(kv.Key)
		if index, ok := dupindex[skey]; ok {
			//重复的key 替换老的key
			kvs[index] = kv
		} else {
			dupindex[skey] = n
			kvs[n] = kv
			n++
		}
	}
	return kvs[0:n]
}
