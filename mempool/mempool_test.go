package mempool

import (
	"testing"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

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

func init() {
	queue.DisableLog()
	//	DisableLog() // 不输出任何log
	//	SetLogLevel("debug") // 输出DBUG(含)以下log
	//	SetLogLevel("info") // 输出INFO(含)以下log
	//	SetLogLevel("warn") // 输出WARN(含)以下log
	//	SetLogLevel("eror") // 输出EROR(含)以下log
	//	SetLogLevel("crit") // 输出CRIT(含)以下log
	SetLogLevel("") // 输出所有log
}

func initEnv(size int) (*Mempool, *queue.Queue) {
	var q = queue.New("channel")
	mem := New()
	mem.SetQueue(q)
	if size > 0 {
		mem.Resize(size)
	}
	return mem, q
}

func TestAddTx(t *testing.T) {
	mem, q := initEnv(0)
	qclient := q.GetClient()

	msg := qclient.NewMessage("mempool", types.EventTx, tx1)
	qclient.Send(msg, true)
	qclient.Wait(msg)
	if mem.Size() != 1 {
		t.Error("TestAddTx failed")
	}
}

func TestAddDuplicatedTx(t *testing.T) {
	mem, q := initEnv(0)
	qclient := q.GetClient()

	msg1 := qclient.NewMessage("mempool", types.EventTx, tx2)
	qclient.Send(msg1, true)
	qclient.Wait(msg1)

	msg2 := qclient.NewMessage("mempool", types.EventTx, tx2)
	qclient.Send(msg2, true)
	qclient.Wait(msg2)

	if mem.Size() != 1 {
		t.Error("TestAddDuplicatedTx failed")
	}
}

func TestAddMempool(t *testing.T) {
	mem, q := initEnv(0)
	qclient := q.GetClient()

	msg := qclient.NewMessage("mempool", types.EventTxAddMempool, tx3)
	qclient.Send(msg, true)
	qclient.Wait(msg)

	if mem.Size() != 1 {
		t.Error("TestAddMempool failed")
	}
}

func TestAddDuplicatedTxToMempool(t *testing.T) {
	mem, q := initEnv(0)
	qclient := q.GetClient()

	msg1 := qclient.NewMessage("mempool", types.EventTxAddMempool, tx4)
	qclient.Send(msg1, true)
	qclient.Wait(msg1)

	msg2 := qclient.NewMessage("mempool", types.EventTxAddMempool, tx4)
	qclient.Send(msg2, true)
	qclient.Wait(msg2)

	if mem.Size() != 1 {
		t.Error("TestAddDuplicatedTxToMempool failed")
	}
}

func add4Tx(qclient queue.IClient) {
	msg1 := qclient.NewMessage("mempool", types.EventTx, tx1)
	msg2 := qclient.NewMessage("mempool", types.EventTx, tx2)
	msg3 := qclient.NewMessage("mempool", types.EventTx, tx3)
	msg4 := qclient.NewMessage("mempool", types.EventTx, tx4)

	qclient.Send(msg1, true)
	qclient.Wait(msg1)

	qclient.Send(msg2, true)
	qclient.Wait(msg2)

	qclient.Send(msg3, true)
	qclient.Wait(msg3)

	qclient.Send(msg4, true)
	qclient.Wait(msg4)
}

func TestGetTxList(t *testing.T) {

	mem, q := initEnv(0)
	qclient := q.GetClient()

	//add tx
	add4Tx(qclient)

	msg := qclient.NewMessage("mempool", types.EventTxList, 100)
	qclient.Send(msg, true)
	data, err := qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	txs := data.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs) != 4 {
		t.Error("get txlist number error")
	}

	if mem.Size() != 0 {
		t.Error("TestGetTxList failed")
	}
}

func TestAddMoreTxThanPoolSize(t *testing.T) {

	mem, q := initEnv(4)
	qclient := q.GetClient()

	add4Tx(qclient)

	msg5 := qclient.NewMessage("mempool", types.EventTx, tx5)
	qclient.Send(msg5, true)
	qclient.Wait(msg5)

	if mem.Size() != 4 || mem.cache.Exists(tx1) {
		t.Error("TestAddMoreTxThanPoolSize failed")
	}
}

func TestRemoveTxOfBlock(t *testing.T) {
	mem, q := initEnv(0)
	qclient := q.GetClient()

	add4Tx(qclient)

	msg5 := qclient.NewMessage("mempool", types.EventAddBlock, blk)
	qclient.Send(msg5, false)

	msg := qclient.NewMessage("mempool", types.EventGetMempoolSize, nil)
	qclient.Send(msg, true)

	mem.SetQueue(q)
	reply, err := qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if reply.GetData().(*types.MempoolSize).Size != 3 {
		t.Error("TestGetMempoolSize failed")
	}
}
