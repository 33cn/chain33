package util

import (
	"bytes"
	"errors"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

var ulog = log.New("module", "util")

//block.Txs empty?
func checkSignWrapper(block *types.Block, client queue.Client) bool {
	var signedByAuth bool = false
	for _, tx := range block.Txs {
		//only one type of signature will be used in all txs
		if tx.GetSignature().GetTy() == types.SIG_TYPE_AUTHORITY {
			signedByAuth = true
			break
		}
	}

	if signedByAuth {
		if !block.CheckBlockSign() {
			return false
		}
		data := &types.ReqAuthSignCheckTxs{
			Txs: block.Txs,
		}

		msg := client.NewMessage("authority", types.EventAuthorityCheckTxs, data)
		client.Send(msg, true)

		resp, err := client.Wait(msg)
		if err != nil {
			panic(err)
		}

		res, ok := resp.GetData().(*types.RespAuthSignCheckTxs)
		if !ok || !res.Result {
			return false
		}
		return true
	} else {
		return block.CheckSign()
	}
}

//block执行函数增加一个批量存储区块是否刷盘的标志位，提高区块的同步性能。
//只有blockchain在同步阶段会设置不刷盘，其余模块处理时默认都是刷盘的
func ExecBlock(client queue.Client, prevStateRoot []byte, block *types.Block, errReturn bool, sync bool) (*types.BlockDetail, []*types.Transaction, error) {
	//发送执行交易给execs模块
	//通过consensus module 再次检查
	ulog.Debug("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
	beg := time.Now()
	defer func() {
		ulog.Info("ExecBlock", "height", block.Height, "ntx", len(block.Txs), "writebatchsync", sync, "cost", time.Since(beg))
	}()

	if errReturn && block.Height > 0 {
		//block的来源不是自己的mempool，而是别人的区块
		if !checkSignWrapper(block, client) {
			return nil, nil, types.ErrSign
		}
	}
	//tx交易去重处理, 这个地方要查询数据库，需要一个更快的办法
	cacheTxs := types.TxsToCache(block.Txs)
	oldtxscount := len(cacheTxs)
	cacheTxs = CheckTxDup(client, cacheTxs, block.Height)
	newtxscount := len(cacheTxs)
	if oldtxscount != newtxscount && errReturn {
		return nil, nil, types.ErrTxDup
	}
	ulog.Debug("ExecBlock", "prevtx", oldtxscount, "newtx", newtxscount)
	block.TxHash = merkle.CalcMerkleRootCache(cacheTxs)
	block.Txs = types.CacheToTxs(cacheTxs)

	receipts := ExecTx(client, prevStateRoot, block)
	var maplist = make(map[string]*types.KeyValue)
	var kvset []*types.KeyValue
	var deltxlist = make(map[int]bool)
	var rdata []*types.ReceiptData //save to db receipt log
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		if receipt.Ty == types.ExecErr {
			ulog.Error("exec tx err", "err", receipt)
			if errReturn { //认为这个是一个错误的区块
				return nil, nil, types.ErrBlockExec
			}
			deltxlist[i] = true
			continue
		}
		rdata = append(rdata, &types.ReceiptData{receipt.Ty, receipt.Logs})
		//处理KV
		kvs := receipt.KV
		for _, kv := range kvs {
			if item, ok := maplist[string(kv.Key)]; ok {
				item.Value = kv.Value //更新item 的value
			} else {
				maplist[string(kv.Key)] = kv
				kvset = append(kvset, kv)
			}
		}
	}
	//check TxHash
	calcHash := merkle.CalcMerkleRoot(block.Txs)
	if errReturn && !bytes.Equal(calcHash, block.TxHash) {
		return nil, nil, types.ErrCheckTxHash
	}
	block.TxHash = calcHash
	//删除无效的交易
	var deltx []*types.Transaction
	if len(deltxlist) > 0 {
		var newtx []*types.Transaction
		for i := 0; i < len(block.Txs); i++ {
			if deltxlist[i] {
				deltx = append(deltx, block.Txs[i])
			} else {
				newtx = append(newtx, block.Txs[i])
			}
		}
		block.Txs = newtx
		block.TxHash = merkle.CalcMerkleRoot(block.Txs)
	}

	var detail types.BlockDetail
	if kvset == nil {
		calcHash = prevStateRoot
	} else {
		calcHash = ExecKVMemSet(client, prevStateRoot, kvset, sync)
	}
	if errReturn && !bytes.Equal(block.StateHash, calcHash) {
		ExecKVSetRollback(client, calcHash)
		if len(rdata) > 0 {
			for _, rd := range rdata {
				rd.OutputReceiptDetails(ulog)
			}
		}
		return nil, nil, types.ErrCheckStateHash
	}
	block.StateHash = calcHash
	detail.Block = block
	detail.Receipts = rdata
	if detail.Block.Height > 0 {
		err := CheckBlock(client, &detail)
		if err != nil {
			ulog.Debug("CheckBlock-->", "err=", err)
			return nil, deltx, err
		}
	}
	//save to db
	if kvset != nil {
		ExecKVSetCommit(client, block.StateHash)
	}

	//get receipts
	//save kvset and get state hash
	//ulog.Debug("blockdetail-->", "detail=", detail)
	return &detail, deltx, nil
}

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
	list := &types.ExecTxList{prevStateRoot, block.Txs, block.BlockTime, block.Height, uint64(block.Difficulty)}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	receipts := resp.GetData().(*types.Receipts)
	return receipts
}

func ExecKVMemSet(client queue.Client, prevStateRoot []byte, kvset []*types.KeyValue, sync bool) []byte {
	set := &types.StoreSet{prevStateRoot, kvset}
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

func CheckTxDup(client queue.Client, txs []*types.TransactionCache, height int64) (transactions []*types.TransactionCache) {
	var checkHashList types.TxHashList
	if height >= types.ForkV1 {
		txs = CheckTxDupInner(txs)
	}
	for _, tx := range txs {
		hash := tx.Hash()
		checkHashList.Hashes = append(checkHashList.Hashes, hash)
	}
	hashList := client.NewMessage("blockchain", types.EventTxHashList, &checkHashList)
	client.Send(hashList, true)
	dupTxList, _ := client.Wait(hashList)
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
	return transactions
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
