package util

import (
	"bytes"
	"time"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var ulog = log.New("module", "util")

func ExecBlock(q *queue.Queue, prevStateRoot []byte, block *types.Block, errReturn bool) (*types.BlockDetail, error) {
	//发送执行交易给execs模块
	ulog.Error("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
	beg := time.Now()
	receipts := ExecTx(q, prevStateRoot, block)
	ulog.Error("ExecBlock", "cost", time.Now().Sub(beg))
	var maplist = make(map[string]*types.KeyValue)
	var kvset []*types.KeyValue
	var deltxlist = make(map[int]bool)
	var rdata []*types.ReceiptData //save to db receipt log
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		if receipt.Ty == types.ExecErr {
			if errReturn { //认为这个是一个错误的区块
				return nil, types.ErrBlockExec
			}
			ulog.Error("exec tx err", "err", receipt)
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
		return nil, types.ErrCheckTxHash
	}
	block.TxHash = calcHash
	//删除无效的交易
	if len(deltxlist) > 0 {
		var newtx []*types.Transaction
		for i := 0; i < len(block.Txs); i++ {
			if !deltxlist[i] {
				newtx = append(newtx, block.Txs[i])
			}
		}
		block.Txs = newtx
		block.TxHash = merkle.CalcMerkleRoot(block.Txs)
	}

	var detail types.BlockDetail
	currentHash := block.StateHash
	if kvset == nil {
		block.StateHash = prevStateRoot
	} else {
		block.StateHash = ExecKVMemSet(q, prevStateRoot, kvset)
	}
	if errReturn && !bytes.Equal(currentHash, block.StateHash) {
		ExecKVSetRollback(q, block.StateHash)
		return nil, types.ErrCheckStateHash
	}
	//save to db
	ExecKVSetCommit(q, block.StateHash)
	detail.Block = block
	detail.Receipts = rdata
	//get receipts
	//save kvset and get state hash
	ulog.Debug("blockdetail-->", "detail=", detail)
	return &detail, nil
}

func ExecTx(q *queue.Queue, prevStateRoot []byte, block *types.Block) *types.Receipts {
	client := q.GetClient()
	list := &types.ExecTxList{prevStateRoot, block.Txs, block.BlockTime, block.Height}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	receipts := resp.GetData().(*types.Receipts)
	return receipts
}

func ExecTxList(q *queue.Queue, prevStateRoot []byte, txs []*types.Transaction, header *types.Header) *types.Receipts {
	client := q.GetClient()
	list := &types.ExecTxList{prevStateRoot, txs, header.BlockTime, header.Height}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	receipts := resp.GetData().(*types.Receipts)
	return receipts
}

func ExecKVMemSet(q *queue.Queue, prevStateRoot []byte, kvset []*types.KeyValue) []byte {
	client := q.GetClient()
	set := &types.StoreSet{prevStateRoot, kvset}
	msg := client.NewMessage("store", types.EventStoreMemSet, set)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	hash := resp.GetData().(*types.ReplyHash)
	return hash.GetHash()
}

func ExecKVSetCommit(q *queue.Queue, hash []byte) error {
	qclient := q.GetClient()
	req := &types.ReqHash{hash}
	msg := qclient.NewMessage("store", types.EventStoreCommit, req)
	qclient.Send(msg, true)
	msg, err := qclient.Wait(msg)
	if err != nil {
		return err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	return nil
}

func ExecKVSetRollback(q *queue.Queue, hash []byte) error {
	qclient := q.GetClient()
	req := &types.ReqHash{hash}
	msg := qclient.NewMessage("store", types.EventStoreRollback, req)
	qclient.Send(msg, true)
	msg, err := qclient.Wait(msg)
	if err != nil {
		return err
	}
	hash = msg.GetData().(*types.ReplyHash).GetHash()
	return nil
}
