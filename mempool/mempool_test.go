package mempool

import (
	"log"
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

var blk = &types.Block{
	Version:    1,
	ParentHash: []byte("parent hash"),
	TxHash:     []byte("tx hash"),
	Height:     1,
	BlockTime:  1,
	Txs:        []*types.Transaction{tx1, tx3},
}

func TestAddTx(t *testing.T) {
	msg := qclient.NewMessage("mempool", types.EventTx, tx1)
	qclient.Send(msg, true)
	mem.SetQueue(q)
	qclient.Wait(msg)
	log.Printf("1. size of mempool: %d", mem.Size())
}

func TestAddDuplicatedTx(t *testing.T) {
	msg1 := qclient.NewMessage("mempool", types.EventTx, tx2)
	msg2 := qclient.NewMessage("mempool", types.EventTx, tx2)

	qclient.Send(msg1, true)
	qclient.Send(msg2, true)

	mem.SetQueue(q)

	qclient.Wait(msg1)
	qclient.Wait(msg2)
	log.Printf("2. size of mempool: %d", mem.Size())
}

func TestAddMempool(t *testing.T) {
	msg := qclient.NewMessage("mempool", types.EventTxAddMempool, tx3)
	qclient.Send(msg, true)
	mem.SetQueue(q)
	qclient.Wait(msg)
	log.Printf("3. size of mempool: %d", mem.Size())
}

func TestAddDuplicatedTxToMempool(t *testing.T) {
	msg1 := qclient.NewMessage("mempool", types.EventTxAddMempool, tx4)
	msg2 := qclient.NewMessage("mempool", types.EventTxAddMempool, tx4)

	qclient.Send(msg1, true)
	qclient.Send(msg2, true)

	mem.SetQueue(q)

	qclient.Wait(msg1)
	qclient.Wait(msg2)
	log.Printf("4. size of mempool: %d", mem.Size())
}

func TestGetTxList(t *testing.T) {
	msg := qclient.NewMessage("mempool", types.EventTxList, 100)
	qclient.Send(msg, true)
	mem.SetQueue(q)
	qclient.Wait(msg)
	log.Printf("5. size of mempool: %d", mem.Size())
}

func TestRemoveTxOfBlock(t *testing.T) {
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

	log.Printf("6. size of mempool: %d", mem.Size())

	msg5 := qclient.NewMessage("mempool", types.EventAddBlock, blk)
	qclient.Send(msg5, false)
	mem.SetQueue(q)
	memSize := mem.Size()
	log.Printf("7. size of mempool: %d", memSize)
}
