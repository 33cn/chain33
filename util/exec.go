package util

import (
	"bytes"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var ulog = log.New("module", "util")

func ExecBlock(q *queue.Queue, prevStateRoot []byte, block *types.Block, errReturn bool) (*types.BlockDetail, error) {
	//发送执行交易给execs模块
	receipts := ExecTx(q, prevStateRoot, block)
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
		block.StateHash = ExecKVSet(q, prevStateRoot, kvset)
	}
	if errReturn && bytes.Equal(currentHash, block.StateHash) {
		return nil, types.ErrCheckStateHash
	}
	detail.Block = block
	detail.Receipts = rdata
	//get receipts
	//save kvset and get state hash
	ulog.Info("blockdetail-->", "detail=", detail)
	return &detail, nil
}

func ExecTx(q *queue.Queue, prevStateRoot []byte, block *types.Block) *types.Receipts {
	client := q.GetClient()
	list := &types.ExecTxList{prevStateRoot, block.Txs}
	msg := client.NewMessage("execs", types.EventExecTxList, list)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	receipts := resp.GetData().(*types.Receipts)
	return receipts
}

func ExecKVSet(q *queue.Queue, prevStateRoot []byte, kvset []*types.KeyValue) []byte {
	client := q.GetClient()
	set := &types.StoreSet{prevStateRoot, kvset}
	msg := client.NewMessage("store", types.EventStoreSet, &set)
	client.Send(msg, true)
	resp, err := client.Wait(msg)
	if err != nil {
		panic(err)
	}
	hash := resp.GetData().(*types.ReplyHash)
	return hash.GetHash()
}
