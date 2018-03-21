package mempool

import (
	"errors"
	"flag"
	"math/rand"
	"testing"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/config"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/common/limits"
	"code.aliyun.com/chain33/chain33/consensus"
	"code.aliyun.com/chain33/chain33/executor"
	"code.aliyun.com/chain33/chain33/p2p"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/store"
	"code.aliyun.com/chain33/chain33/types"
)

//----------------------------- data for testing ---------------------------------
var (
	amount   = int64(1e8)
	v        = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx1      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 10}
	tx2      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100000000, Expire: 0}
	tx3      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 200000000, Expire: 0}
	tx4      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 300000000, Expire: 0}
	tx5      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 400000000, Expire: 0}
	tx6      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 500000000, Expire: 0}
	tx7      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 600000000, Expire: 0}
	tx8      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 700000000, Expire: 0}
	tx9      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 800000000, Expire: 0}
	tx10     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 900000000, Expire: 0}
	tx11     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 450000000, Expire: 0}
	tx12     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 460000000, Expire: 0}

	c, _       = crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	hex        = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	a, _       = common.FromHex(hex)
	privKey, _ = c.PrivKeyFromBytes(a)
	random     *rand.Rand
	mainPriv   crypto.PrivKey
)

//var privTo, _ = c.GenKey()
//var ad = account.PubKeyToAddress(privKey.PubKey().Bytes()).String()

var blk = &types.Block{
	Version:    1,
	ParentHash: []byte("parent hash"),
	TxHash:     []byte("tx hash"),
	Height:     1,
	BlockTime:  1,
	Txs:        []*types.Transaction{tx3, tx5},
}

func init() {
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(time.Now().UnixNano()))
	queue.DisableLog()
	DisableLog() // 不输出任何log
	//	SetLogLevel("debug") // 输出DBUG(含)以下log
	//	SetLogLevel("info") // 输出INFO(含)以下log
	SetLogLevel("error") // 输出WARN(含)以下log
	//	SetLogLevel("eror") // 输出EROR(含)以下log
	//	SetLogLevel("crit") // 输出CRIT(含)以下log
	//	SetLogLevel("") // 输出所有log
	//	maxTxNumPerAccount = 10000
	mainPriv = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

func initEnv2(size int) (*Mempool, *queue.Queue, *blockchain.BlockChain, *store.Store, *p2p.P2p) {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("chain33.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueue(q)

	exec := executor.New()
	exec.SetQueue(q)

	s := store.New(cfg.Store)
	s.SetQueue(q)

	cs := consensus.New(cfg.Consensus)
	cs.SetQueue(q)

	mem := New(cfg.MemPool)
	mem.SetQueue(q)

	network := p2p.New(cfg.P2P)
	network.SetQueue(q)

	if size > 0 {
		mem.Resize(size)
	}
	mem.SetMinFee(0)
	return mem, q, chain, s, network
}

func initEnv(size int) (*Mempool, *queue.Queue, *blockchain.BlockChain, *store.Store) {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("chain33.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueue(q)

	exec := executor.New()
	exec.SetQueue(q)

	s := store.New(cfg.Store)
	s.SetQueue(q)

	cs := consensus.New(cfg.Consensus)
	cs.SetQueue(q)

	mem := New(cfg.MemPool)
	mem.SetQueue(q)
	mem.sync = true

	if size > 0 {
		mem.Resize(size)
	}

	mem.SetMinFee(types.MinFee)

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
	tx12.Sign(types.SECP256K1, privKey)

	return mem, q, chain, s
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	tx.Nonce = rand.Int63()
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func genaddress() (string, crypto.PrivKey) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		panic(err)
	}
	privto, err := cr.GenKey()
	if err != nil {
		panic(err)
	}
	addrto := account.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func TestAddTx(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	msg := mem.qclient.NewMessage("mempool", types.EventTx, tx1)
	mem.qclient.Send(msg, true)
	mem.qclient.Wait(msg)

	if mem.Size() != 1 {
		t.Error("TestAddTx failed")
	}

	chain.Close()
	s.Close()
	mem.Close()
}

func TestAddDuplicatedTx(t *testing.T) {
	mem, _, chain, s := initEnv(0)
	defer func() {
		chain.Close()
		s.Close()
		mem.Close()
	}()
	msg1 := mem.qclient.NewMessage("mempool", types.EventTx, tx2)
	err := mem.qclient.Send(msg1, true)
	if err != nil {
		t.Error(err)
		return
	}
	msg1, err = mem.qclient.Wait(msg1)
	if err != nil {
		t.Error(err)
		return
	}
	reply := msg1.GetData().(*types.Reply)
	err = checkReply(reply)
	if err != nil {
		t.Error(err)
		return
	}
	if mem.Size() != 1 {
		t.Error("TestAddDuplicatedTx failed", "size", mem.Size())
	}
	msg2 := mem.qclient.NewMessage("mempool", types.EventTx, tx2)
	mem.qclient.Send(msg2, true)
	mem.qclient.Wait(msg2)

	if mem.Size() != 1 {
		t.Error("TestAddDuplicatedTx failed", "size", mem.Size())
	}
}

func checkReply(reply *types.Reply) error {
	if !reply.GetIsOk() {
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func add4Tx(qclient queue.Client) {
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

func add10Tx(qclient queue.Client) {
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
	mem, _, chain, s := initEnv(0)

	// add tx
	add4Tx(mem.qclient)

	msg1 := mem.qclient.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 2, Hashes: nil})
	mem.qclient.Send(msg1, true)
	data1, err := mem.qclient.Wait(msg1)
	if err != nil {
		t.Error(err)
		return
	}
	txs1 := data1.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs1) != 2 {
		t.Error("get txlist number error")
	}

	var hashList [][]byte
	for _, tx := range txs1 {
		hashList = append(hashList, tx.Hash())
	}
	msg2 := mem.qclient.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 2, Hashes: hashList})
	mem.qclient.Send(msg2, true)
	data2, err := mem.qclient.Wait(msg2)
	if err != nil {
		t.Error(err)
		return
	}
	txs2 := data2.GetData().(*types.ReplyTxList).GetTxs()
OutsideLoop:
	for _, t1 := range txs1 {
		for _, t2 := range txs2 {
			if string(t1.Hash()) == string(t2.Hash()) {
				t.Error("TestGetTxList failed")
				break OutsideLoop
			}
		}
	}

	chain.Close()
	s.Close()
	mem.Close()
}

func TestAddMoreTxThanPoolSize(t *testing.T) {
	mem, _, chain, s := initEnv(4)

	add4Tx(mem.qclient)

	msg5 := mem.qclient.NewMessage("mempool", types.EventTx, tx5)
	mem.qclient.Send(msg5, true)
	mem.qclient.Wait(msg5)

	if mem.Size() != 4 || mem.cache.Exists(tx5) {
		t.Error("TestAddMoreTxThanPoolSize failed")
	}

	chain.Close()
	s.Close()
	mem.Close()
}

func TestRemoveTxOfBlock(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	add4Tx(mem.qclient)

	blkDetail := &types.BlockDetail{Block: blk}
	msg5 := mem.qclient.NewMessage("mempool", types.EventAddBlock, blkDetail)
	mem.qclient.Send(msg5, false)

	msg := mem.qclient.NewMessage("mempool", types.EventGetMempoolSize, nil)
	mem.qclient.Send(msg, true)

	reply, err := mem.qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if reply.GetData().(*types.MempoolSize).Size != 3 {
		t.Error("TestGetMempoolSize failed")
	}

	chain.Close()
	s.Close()
	mem.Close()
}

func TestDuplicateMempool(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// add 10 txs
	add10Tx(mem.qclient)

	msg := mem.qclient.NewMessage("mempool", types.EventGetMempool, nil)
	mem.qclient.Send(msg, true)

	reply, err := mem.qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if len(reply.GetData().(*types.ReplyTxList).GetTxs()) != 10 || mem.Size() != 10 {
		t.Error("TestDuplicateMempool failed")
	}

	chain.Close()
	s.Close()
	mem.Close()
}

func TestGetLatestTx(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// add 10 txs
	add10Tx(mem.qclient)

	msg := mem.qclient.NewMessage("mempool", types.EventGetLastMempool, nil)
	mem.qclient.Send(msg, true)

	reply, err := mem.qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if len(reply.GetData().(*types.ReplyTxList).GetTxs()) != 10 || mem.Size() != 10 {
		t.Error("TestGetLatestTx failed")
	}

	chain.Close()
	s.Close()
	mem.Close()
}

//func TestBigMessage(t *testing.T) {
//	mem, q, chain := initEnv(0)
//	mem.qclient := q.NewClient()
//}

func TestCheckLowFee(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	mem.SetMinFee(1000)
	tmp := *tx11
	copytx := &tmp
	copytx.Fee = 100 // make low tx fee
	copytx.Sign(types.SECP256K1, privKey)
	msg := mem.qclient.NewMessage("mempool", types.EventTx, copytx)
	mem.qclient.Send(msg, true)
	resp, _ := mem.qclient.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrTxFeeTooLow.Error() {
		t.Error("TestCheckLowFee failed")
	}

	chain.Close()
	s.Close()
	mem.Close()
}

//func TestCheckManyTxs(t *testing.T) {
//	mem, _, chain, s := initEnv(0)

//	// add 10 txs for the same account
//	add10Tx(mem.qclient)

//	msg11 := mem.qclient.NewMessage("mempool", types.EventTx, tx11)
//	mem.qclient.Send(msg11, true)
//	resp, _ := mem.qclient.Wait(msg11)

//	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrManyTx.Error() || mem.Size() != int(maxTxNumPerAccount) {
//		t.Error("TestCheckManyTxs failed")
//	}

//	chain.Close()
//	s.Close()
//	mem.Close()
//}

func TestCheckSignature(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// make wrong signature
	tx12.Signature.Signature = tx12.Signature.Signature[5:]

	msg := mem.qclient.NewMessage("mempool", types.EventTx, tx12)
	mem.qclient.Send(msg, true)
	resp, _ := mem.qclient.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrSign.Error() {
		t.Error("TestCheckSignature failed", string(resp.GetData().(*types.Reply).GetMsg()))
	}

	chain.Close()
	s.Close()
	mem.Close()
}

//func TestCheckBalance(t *testing.T) {
//	mem, _, chain, s := initEnv(0)
//	tmp := *tx11
//	copytx := &tmp
//	copytx.Fee = 999999999999999999
//	copytx.Sign(types.SECP256K1, privKey)
//	msg := mem.qclient.NewMessage("mempool", types.EventTx, copytx)
//	mem.qclient.Send(msg, true)
//	resp, _ := mem.qclient.Wait(msg)
//	println(resp.GetData())
//	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrBalanceLessThanTenTimesFee.Error() {
//		t.Error("TestCheckBalance failed", resp.GetData().(*types.Reply))
//	}

//	chain.Close()
//	s.Close()
//	mem.Close()
//}

func TestCheckExpire1(t *testing.T) {
	mem, _, chain, s := initEnv(0)
	mem.header = &types.Header{Height: 50, BlockTime: 1e9 + 1}
	tmp := *tx11
	copytx := &tmp
	copytx.SetExpire(48) // make tx expired
	copytx.Sign(types.SECP256K1, privKey)
	msg := mem.qclient.NewMessage("mempool", types.EventTx, copytx)
	mem.qclient.Send(msg, true)
	resp, _ := mem.qclient.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrTxExpire.Error() {
		t.Error("TestCheckExpire failed", string(resp.GetData().(*types.Reply).GetMsg()))
	}

	chain.Close()
	s.Close()
	mem.Close()
}

func TestCheckExpire2(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// add tx
	add4Tx(mem.qclient)

	mem.header = &types.Header{Height: 50, BlockTime: 1e9 + 1}

	msg := mem.qclient.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 100, Hashes: nil})
	mem.qclient.Send(msg, true)
	data, err := mem.qclient.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	txs := data.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs) != 3 {
		t.Error("TestCheckExpire failed")
	}

	chain.Close()
	s.Close()
	mem.Close()
}

func BenchmarkMempool(b *testing.B) {
	maxTxNumPerAccount = 100000
	mem, _, chain, s := initEnv(0)
	for i := 0; i < b.N; i++ {
		to, _ := genaddress()
		tx := createTx(mainPriv, to, 10000)
		msg := mem.qclient.NewMessage("mempool", types.EventTx, tx)
		err := mem.qclient.Send(msg, true)
		if err != nil {
			println(err)
		}
	}
	to0, _ := genaddress()
	tx0 := createTx(mainPriv, to0, 10000)
	msg := mem.qclient.NewMessage("mempool", types.EventTx, tx0)
	mem.qclient.Send(msg, true)
	mem.qclient.Wait(msg)
	println(mem.Size() == b.N+1)

	chain.Close()
	s.Close()
	mem.Close()
}
