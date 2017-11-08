package mempool

import (
	"testing"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

var mem = New()
var q = queue.New("channel")
var qclient = q.GetClient()

var tx1 = &types.Transaction{Account: []byte("tester1"), Payload: []byte("mempool"), Signature: []byte("2017110101")}
var tx2 = &types.Transaction{Account: []byte("tester2"), Payload: []byte("mempool"), Signature: []byte("2017110102")}
var tx3 = &types.Transaction{Account: []byte("tester3"), Payload: []byte("mempool"), Signature: []byte("2017110103")}
var tx4 = &types.Transaction{Account: []byte("tester4"), Payload: []byte("mempool"), Signature: []byte("2017110104")}
var tx5 = &types.Transaction{Account: []byte("tester5"), Payload: []byte("mempool"), Signature: []byte("2017110105")}

var blk = &types.Block{
	Version:    1,
	ParentHash: []byte("parent hash"),
	TxHash:     []byte("tx hash"),
	Height:     1,
	BlockTime:  1,
	Txs:        []*types.Transaction{tx3, tx5},
}

func TestSetMempoolSize(t *testing.T) {
	mem.cache.SetMempoolSize(4)
	if mem.cache.size != 4 {
		t.Error("TestSetMempoolSize failed")
	}
}

func TestAddTx(t *testing.T) {
	msg := qclient.NewMessage("mempool", types.EventTx, tx1)
	qclient.Send(msg, true)
	mem.SetQueue(q)
	qclient.Wait(msg)

	if mem.Size() != 1 {
		t.Error("TestAddTx failed")
	}
}

func TestAddDuplicatedTx(t *testing.T) {
	msg1 := qclient.NewMessage("mempool", types.EventTx, tx2)
	msg2 := qclient.NewMessage("mempool", types.EventTx, tx2)

	qclient.Send(msg1, true)
	qclient.Send(msg2, true)

	mem.SetQueue(q)

	qclient.Wait(msg1)
	qclient.Wait(msg2)

	if mem.Size() != 2 {
		t.Error("TestAddDuplicatedTx failed")
	}
}

func TestAddMempool(t *testing.T) {
	msg := qclient.NewMessage("mempool", types.EventTxAddMempool, tx3)
	qclient.Send(msg, true)
	mem.SetQueue(q)
	qclient.Wait(msg)

	if mem.Size() != 3 {
		t.Error("TestAddMempool failed")
	}
}

func TestAddDuplicatedTxToMempool(t *testing.T) {
	msg1 := qclient.NewMessage("mempool", types.EventTxAddMempool, tx4)
	msg2 := qclient.NewMessage("mempool", types.EventTxAddMempool, tx4)

	qclient.Send(msg1, true)
	qclient.Send(msg2, true)

	mem.SetQueue(q)

	qclient.Wait(msg1)
	qclient.Wait(msg2)

	if mem.Size() != 4 {
		t.Error("TestAddDuplicatedTxToMempool failed")
	}
}

func TestGetTxList(t *testing.T) {
	msg := qclient.NewMessage("mempool", types.EventTxList, 100)
	qclient.Send(msg, true)
	mem.SetQueue(q)
	_, err := qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if mem.Size() != 0 {
		t.Error("TestGetTxList failed")
	}
}

func TestAddMoreTxThanPoolSize(t *testing.T) {
	msg1 := qclient.NewMessage("mempool", types.EventTx, tx1)
	msg2 := qclient.NewMessage("mempool", types.EventTx, tx2)
	msg3 := qclient.NewMessage("mempool", types.EventTx, tx3)
	msg4 := qclient.NewMessage("mempool", types.EventTx, tx4)

	qclient.Send(msg1, true)
	qclient.Send(msg2, true)
	qclient.Send(msg3, true)
	qclient.Send(msg4, true)

	mem.SetQueue(q)

	qclient.Wait(msg1)
	qclient.Wait(msg2)
	qclient.Wait(msg3)
	qclient.Wait(msg4)

	msg5 := qclient.NewMessage("mempool", types.EventTx, tx5)
	qclient.Send(msg5, true)
	mem.SetQueue(q)
	qclient.Wait(msg5)

	if mem.Size() != 4 || mem.cache.Exists(tx1) {
		t.Error("TestAddMoreTxThanPoolSize failed")
	}
}

func TestRemoveTxOfBlock(t *testing.T) {
	msg5 := qclient.NewMessage("mempool", types.EventAddBlock, blk)
	qclient.Send(msg5, false)
	mem.SetQueue(q)
	TestGetMempoolSize(t)
}

func TestGetMempoolSize(t *testing.T) {
	msg := qclient.NewMessage("mempool", types.EventGetMempoolSize, nil)
	qclient.Send(msg, true)
	mem.SetQueue(q)
	reply, err := qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if reply.GetData().(*types.MempoolSize).Size != 2 {
		t.Error("TestGetMempoolSize failed")
	}
}
