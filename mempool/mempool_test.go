package mempool

import (
	"errors"
	"flag"
	"math/rand"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/consensus"
	"gitlab.33.cn/chain33/chain33/executor"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
)

//----------------------------- data for testing ---------------------------------
var (
	amount   = int64(1e8)
	v        = &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer = &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx1      = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1000000, Expire: 1}
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
	tx13     = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 100, Expire: 0}

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

func initEnv2(size int) (*Mempool, queue.Queue, *blockchain.BlockChain, queue.Module, *p2p.P2p) {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("chain33.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())

	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	cs := consensus.New(cfg.Consensus)
	cs.SetQueueClient(q.Client())

	mem := New(cfg.MemPool)
	mem.SetQueueClient(q.Client())
	mem.setSync(true)

	network := p2p.New(cfg.P2P)
	network.SetQueueClient(q.Client())

	if size > 0 {
		mem.Resize(size)
	}
	mem.SetMinFee(0)
	return mem, q, chain, s, network
}

func initEnv(size int) (*Mempool, queue.Queue, *blockchain.BlockChain, queue.Module) {
	var q = queue.New("channel")
	flag.Parse()
	cfg := config.InitCfg("chain33.toml")
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())

	exec := executor.New(cfg.Exec)
	exec.SetQueueClient(q.Client())

	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())

	cs := consensus.New(cfg.Consensus)
	cs.SetQueueClient(q.Client())

	mem := New(cfg.MemPool)
	mem.SetQueueClient(q.Client())
	mem.setSync(true)

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
	tx13.Sign(types.SECP256K1, privKey)

	return mem, q, chain, s
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{Transfer: &types.CoinsTransfer{Amount: amount}}
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

	msg := mem.client.NewMessage("mempool", types.EventTx, tx2)
	mem.client.Send(msg, true)
	mem.client.Wait(msg)

	if mem.Size() != 1 {
		t.Error("TestAddTx failed")
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestAddDuplicatedTx(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	defer func() {
		s.Close()
		mem.Close()
		chain.Close()
	}()
	msg1 := mem.client.NewMessage("mempool", types.EventTx, tx2)
	err := mem.client.Send(msg1, true)
	if err != nil {
		t.Error(err)
		return
	}
	msg1, err = mem.client.Wait(msg1)
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
	msg2 := mem.client.NewMessage("mempool", types.EventTx, tx2)
	mem.client.Send(msg2, true)
	mem.client.Wait(msg2)

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

func add4Tx(client queue.Client) error {
	msg1 := client.NewMessage("mempool", types.EventTx, tx1)
	msg2 := client.NewMessage("mempool", types.EventTx, tx2)
	msg3 := client.NewMessage("mempool", types.EventTx, tx3)
	msg4 := client.NewMessage("mempool", types.EventTx, tx4)
	client.Send(msg1, true)
	_, err := client.Wait(msg1)
	if err != nil {
		return err
	}

	client.Send(msg2, true)
	_, err = client.Wait(msg2)
	if err != nil {
		return err
	}

	client.Send(msg3, true)
	_, err = client.Wait(msg3)
	if err != nil {
		return err
	}

	client.Send(msg4, true)
	_, err = client.Wait(msg4)
	if err != nil {
		return err
	}
	return nil
}

func add4TxHash(client queue.Client) ([]string, error) {
	msg1 := client.NewMessage("mempool", types.EventTx, tx5)
	msg2 := client.NewMessage("mempool", types.EventTx, tx2)
	msg3 := client.NewMessage("mempool", types.EventTx, tx3)
	msg4 := client.NewMessage("mempool", types.EventTx, tx4)
	hashList := []string{string(tx5.Hash()), string(tx2.Hash()), string(tx3.Hash()), string(tx4.Hash())}
	client.Send(msg1, true)
	_, err := client.Wait(msg1)
	if err != nil {
		return nil, err
	}

	client.Send(msg2, true)
	_, err = client.Wait(msg2)
	if err != nil {
		return nil, err
	}

	client.Send(msg3, true)
	_, err = client.Wait(msg3)
	if err != nil {
		return nil, err
	}

	client.Send(msg4, true)
	_, err = client.Wait(msg4)
	if err != nil {
		return nil, err
	}
	return hashList, nil
}

func add10Tx(client queue.Client) error {
	err := add4Tx(client)
	if err != nil {
		return err
	}

	msg5 := client.NewMessage("mempool", types.EventTx, tx5)
	msg6 := client.NewMessage("mempool", types.EventTx, tx6)
	msg7 := client.NewMessage("mempool", types.EventTx, tx7)
	msg8 := client.NewMessage("mempool", types.EventTx, tx8)
	msg9 := client.NewMessage("mempool", types.EventTx, tx9)
	msg10 := client.NewMessage("mempool", types.EventTx, tx10)

	client.Send(msg5, true)
	_, err = client.Wait(msg5)
	if err != nil {
		return err
	}

	client.Send(msg6, true)
	_, err = client.Wait(msg6)
	if err != nil {
		return err
	}

	client.Send(msg7, true)
	_, err = client.Wait(msg7)
	if err != nil {
		return err
	}

	client.Send(msg8, true)
	_, err = client.Wait(msg8)
	if err != nil {
		return err
	}

	client.Send(msg9, true)
	_, err = client.Wait(msg9)
	if err != nil {
		return err
	}

	client.Send(msg10, true)
	_, err = client.Wait(msg10)
	if err != nil {
		return err
	}

	return nil
}

func TestGetTxList(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// add tx
	hashes, err := add4TxHash(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	msg1 := mem.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 2, Hashes: nil})
	mem.client.Send(msg1, true)
	data1, err := mem.client.Wait(msg1)
	if err != nil {
		t.Error(err)
		return
	}
	txs1 := data1.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs1) != 2 {
		t.Error("get txlist number error")
	}

	var hashList [][]byte
	for i, tx := range txs1 {
		hashList = append(hashList, tx.Hash())
		if hashes[i] != string(tx.Hash()) {
			t.Error("gettxlist not in time order1")
		}
	}
	msg2 := mem.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 1, Hashes: hashList})
	mem.client.Send(msg2, true)
	data2, err := mem.client.Wait(msg2)
	if err != nil {
		t.Error(err)
		return
	}
	txs2 := data2.GetData().(*types.ReplyTxList).GetTxs()
	for i, tx := range txs2 {
		hashList = append(hashList, tx.Hash())
		if hashes[2+i] != string(tx.Hash()) {
			t.Error("gettxlist not in time order2")
		}
	}
OutsideLoop:
	for _, t1 := range txs1 {
		for _, t2 := range txs2 {
			if string(t1.Hash()) == string(t2.Hash()) {
				t.Error("TestGetTxList failed")
				break OutsideLoop
			}
		}
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestAddMoreTxThanPoolSize(t *testing.T) {
	mem, _, chain, s := initEnv(4)

	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	msg5 := mem.client.NewMessage("mempool", types.EventTx, tx5)
	mem.client.Send(msg5, true)
	mem.client.Wait(msg5)

	if mem.Size() != 4 || mem.cache.Exists(tx5.Hash()) {
		t.Error("TestAddMoreTxThanPoolSize failed")
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestRemoveTxOfBlock(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	blkDetail := &types.BlockDetail{Block: blk}
	msg5 := mem.client.NewMessage("mempool", types.EventAddBlock, blkDetail)
	mem.client.Send(msg5, false)

	msg := mem.client.NewMessage("mempool", types.EventGetMempoolSize, nil)
	mem.client.Send(msg, true)

	reply, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if reply.GetData().(*types.MempoolSize).Size != 3 {
		t.Error("TestGetMempoolSize failed")
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestDuplicateMempool(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// add 10 txs
	err := add10Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	msg := mem.client.NewMessage("mempool", types.EventGetMempool, nil)
	mem.client.Send(msg, true)

	reply, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if len(reply.GetData().(*types.ReplyTxList).GetTxs()) != 10 || mem.Size() != 10 {
		t.Error("TestDuplicateMempool failed")
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestGetLatestTx(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// add 10 txs
	err := add10Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	msg := mem.client.NewMessage("mempool", types.EventGetLastMempool, nil)
	mem.client.Send(msg, true)

	reply, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	if len(reply.GetData().(*types.ReplyTxList).GetTxs()) != 10 || mem.Size() != 10 {
		t.Error("TestGetLatestTx failed", len(reply.GetData().(*types.ReplyTxList).GetTxs()), mem.Size())
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestCheckLowFee(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	mem.SetMinFee(1000)
	msg := mem.client.NewMessage("mempool", types.EventTx, tx13)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrTxFeeTooLow.Error() {
		t.Error("TestCheckLowFee failed")
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestCheckSignature(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// make wrong signature
	tx12.Signature.Signature = tx12.Signature.Signature[5:]

	msg := mem.client.NewMessage("mempool", types.EventTx, tx12)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrSign.Error() {
		t.Error("TestCheckSignature failed", string(resp.GetData().(*types.Reply).GetMsg()))
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestCheckExpire1(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	mem.setHeader(&types.Header{Height: 50, BlockTime: 1e9 + 1})
	msg := mem.client.NewMessage("mempool", types.EventTx, tx1)
	mem.client.Send(msg, true)
	resp, _ := mem.client.Wait(msg)

	if string(resp.GetData().(*types.Reply).GetMsg()) != types.ErrTxExpire.Error() {
		t.Error("TestCheckExpire failed", string(resp.GetData().(*types.Reply).GetMsg()))
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func TestCheckExpire2(t *testing.T) {
	mem, _, chain, s := initEnv(0)

	// add tx
	err := add4Tx(mem.client)
	if err != nil {
		t.Error("add tx error", err.Error())
		return
	}

	mem.setHeader(&types.Header{Height: 50, BlockTime: 1e9 + 1})
	msg := mem.client.NewMessage("mempool", types.EventTxList, &types.TxHashList{Count: 100, Hashes: nil})
	mem.client.Send(msg, true)
	data, err := mem.client.Wait(msg)

	if err != nil {
		t.Error(err)
		return
	}

	txs := data.GetData().(*types.ReplyTxList).GetTxs()

	if len(txs) != 3 {
		t.Error("TestCheckExpire failed", len(txs))
	}

	s.Close()
	mem.Close()
	chain.Close()
}

func BenchmarkMempool(b *testing.B) {
	maxTxNumPerAccount = 100000
	mem, _, chain, s := initEnv(0)
	for i := 0; i < b.N; i++ {
		to, _ := genaddress()
		tx := createTx(mainPriv, to, 10000)
		msg := mem.client.NewMessage("mempool", types.EventTx, tx)
		err := mem.client.Send(msg, true)
		if err != nil {
			println(err)
		}
	}
	to0, _ := genaddress()
	tx0 := createTx(mainPriv, to0, 10000)
	msg := mem.client.NewMessage("mempool", types.EventTx, tx0)
	mem.client.Send(msg, true)
	mem.client.Wait(msg)
	println(mem.Size() == b.N+1)

	s.Close()
	mem.Close()
	chain.Close()
}
