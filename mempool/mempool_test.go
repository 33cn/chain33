package mempool

import (
	"testing"

	//	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/consensus"
	"code.aliyun.com/chain33/chain33/execs"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/store"
	"code.aliyun.com/chain33/chain33/types"
)

var amount = int64(1e8)

var v = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
var transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
var tx1 = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 0}

//&types.Transaction{Execer: []byte("tester1"), Payload: []byte("mempool"), Fee: 20000000, Expire: 0}
var tx2 = &types.Transaction{Execer: []byte("tester2"), Payload: []byte("mempool"), Fee: 40000000, Expire: 0}
var tx3 = &types.Transaction{Execer: []byte("tester3"), Payload: []byte("mempool"), Fee: 10000000, Expire: 0}
var tx4 = &types.Transaction{Execer: []byte("tester4"), Payload: []byte("mempool"), Fee: 30000000, Expire: 0}
var tx5 = &types.Transaction{Execer: []byte("tester5"), Payload: []byte("mempool"), Fee: 50000000, Expire: 0}
var tx6 = &types.Transaction{Execer: []byte("tester6"), Payload: []byte("mempool"), Fee: 50000000, Expire: 0}
var tx7 = &types.Transaction{Execer: []byte("tester7"), Payload: []byte("mempool"), Fee: 10000000, Expire: 0}
var tx8 = &types.Transaction{Execer: []byte("tester8"), Payload: []byte("mempool"), Fee: 30000000, Expire: 0}
var tx9 = &types.Transaction{Execer: []byte("tester9"), Payload: []byte("mempool"), Fee: 50000000, Expire: 0}
var tx10 = &types.Transaction{Execer: []byte("tester10"), Payload: []byte("mempool"), Fee: 20000000, Expire: 0}
var tx11 = &types.Transaction{Execer: []byte("tester11"), Payload: []byte("mempool"), Fee: 10000000, Expire: 0}
var tx12 = &types.Transaction{Execer: []byte("tester12"), Payload: []byte("mempool"), Fee: 10000000, Expire: 0}

var c, _ = crypto.New(types.GetSignatureTypeName(types.SECP256K1))
var hex = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
var privKey, _ = c.PrivKeyFromBytes(common.FromHex(hex))

//var privTo, _ = c.GenKey()
//var ad = account.PubKeyToAddress(privKey.PubKey().Bytes()).String()
//var acc1 = &types.Account{Currency: 1, Balance: 1000000000, Frozen: 0, Addr: ad}

var blk = &types.Block{
	Version:    1,
	ParentHash: []byte("parent hash"),
	TxHash:     []byte("tx hash"),
	Height:     1,
	BlockTime:  1,
	Txs:        []*types.Transaction{tx3, tx5},
}

func init() {
	//	queue.DisableLog()
	//	DisableLog() // 不输出任何log
	//	SetLogLevel("debug") // 输出DBUG(含)以下log
	//	SetLogLevel("info") // 输出INFO(含)以下log
	//	SetLogLevel("warn") // 输出WARN(含)以下log
	//	SetLogLevel("eror") // 输出EROR(含)以下log
	//	SetLogLevel("crit") // 输出CRIT(含)以下log
	SetLogLevel("") // 输出所有log
}

func initEnv(size int) (*Mempool, *queue.Queue, *blockchain.BlockChain) {
	var q = queue.New("channel")
	chain := blockchain.New()
	chain.SetQueue(q)
	exec := execs.New()
	exec.SetQueue(q)
	s := store.New()
	s.SetQueue(q)
	cs := consensus.New()
	cs.SetQueue(q)
	mem := New()
	mem.SetQueue(q)

	if size > 0 {
		mem.Resize(size)
	}

	mem.SetMinFee(0)

	tx1.Sign(types.SECP256K1, privKey)
	tx2.Sign(types.SECP256K1, privKey)
	tx3.Sign(types.SECP256K1, privKey)
	tx4.Sign(types.SECP256K1, privKey)
	tx5.Sign(types.SECP256K1, privKey)
	tx6.Sign(types.SECP256K1, privKey)
	tx7.Sign(types.SECP256K1, privKey)
	tx8.Sign(types.SECP256K1, privKey)
	tx9.Sign(types.SECP256K1, privKey)
	tx10.Sign(types.SECP256K1, privKey)
	tx11.Sign(types.SECP256K1, privKey)

	return mem, q, chain
}

func TestAddTx(t *testing.T) {
	mem, q, chain := initEnv(0)
	qclient := q.GetClient()

	msg := qclient.NewMessage("mempool", types.EventTx, tx1)
	qclient.Send(msg, true)
	qclient.Wait(msg)

	if mem.Size() != 1 {
		t.Error("TestAddTx failed")
	}

	chain.Close()
}

func TestAddDuplicatedTx(t *testing.T) {
	mem, q, chain := initEnv(0)
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

	chain.Close()
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

func add10Tx(qclient queue.IClient) {
	add4Tx(qclient)

	msg5 := qclient.NewMessage("mempool", types.EventTx, tx5)
	msg6 := qclient.NewMessage("mempool", types.EventTx, tx6)
	msg7 := qclient.NewMessage("mempool", types.EventTx, tx7)
	msg8 := qclient.NewMessage("mempool", types.EventTx, tx8)
	msg9 := qclient.NewMessage("mempool", types.EventTx, tx9)
	msg10 := qclient.NewMessage("mempool", types.EventTx, tx10)

	qclient.Send(msg5, true)
	qclient.Wait(msg5)

	qclient.Send(msg6, true)
	qclient.Wait(msg6)

	qclient.Send(msg7, true)
	qclient.Wait(msg7)

	qclient.Send(msg8, true)
	qclient.Wait(msg8)

	qclient.Send(msg9, true)
	qclient.Wait(msg9)

	qclient.Send(msg10, true)
	qclient.Wait(msg10)
}

func TestGetTxList(t *testing.T) {
	mem, q, chain := initEnv(0)
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

	chain.Close()
}

func TestAddMoreTxThanPoolSize(t *testing.T) {
	mem, q, chain := initEnv(4)
	qclient := q.GetClient()

	add4Tx(qclient)

	msg5 := qclient.NewMessage("mempool", types.EventTx, tx5)
	qclient.Send(msg5, true)
	qclient.Wait(msg5)

	if mem.Size() != 4 || mem.cache.Exists(tx3) {
		t.Error("TestAddMoreTxThanPoolSize failed")
	}

	chain.Close()
}

func TestRemoveTxOfBlock(t *testing.T) {
	mem, q, chain := initEnv(0)
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

	chain.Close()
}

//func TestBigMessage(t *testing.T) {
//	mem, q, chain := initEnv(0)
//	qclient := q.GetClient()
//}

func TestCheckLowFee(t *testing.T) {
	mem, q, chain := initEnv(0)
	qclient := q.GetClient()

	mem.SetMinFee(1000)

	tx11.Fee = 100
	msg := qclient.NewMessage("mempool", types.EventTx, tx11)
	qclient.Send(msg, true)
	_, err := qclient.Wait(msg)

	if err.Error() != e02 {
		t.Error("TestCheckLowFee failed")
	}

	chain.Close()
}

func TestCheckManyTxs(t *testing.T) {
	_, q, chain := initEnv(0)
	qclient := q.GetClient()

	add10Tx(qclient)

	msg11 := qclient.NewMessage("mempool", types.EventTx, tx11)
	qclient.Send(msg11, true)
	_, err := qclient.Wait(msg11)

	if err.Error() != e03 {
		t.Error("TestCheckManyTxs failed")
	}

	chain.Close()
}

func TestCheckSignature(t *testing.T) {
	_, q, chain := initEnv(0)
	qclient := q.GetClient()

	msg := qclient.NewMessage("mempool", types.EventTx, tx12)
	qclient.Send(msg, true)
	_, err := qclient.Wait(msg)

	if err.Error() != e04 {
		t.Error("TestCheckSignature failed")
	}

	chain.Close()
}

//func TestCheckBalance(t *testing.T) {
//	mem, q, chain := initEnv(0)
//	qclient := q.GetClient()
//}

func TestCheckExpire(t *testing.T) {
	_, q, chain := initEnv(4)
	qclient := q.GetClient()

	tx11.Expire = 1
	msg := qclient.NewMessage("mempool", types.EventTx, tx11)
	qclient.Send(msg, true)
	_, err := qclient.Wait(msg)

	if err.Error() != e07 {
		t.Error("TestCheckExpire failed")
	}

	chain.Close()
}
