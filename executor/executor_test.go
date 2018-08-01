package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"

	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

var random *rand.Rand
var zeroHash [32]byte
var cfg *types.Config
var genkey crypto.PrivKey

func init() {
	types.SetTitle("local")
	types.SetForkToOne()
	types.SetTestNet(true)
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	random = rand.New(rand.NewSource(types.Now().UnixNano()))
	genkey = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	log.SetLogLevel("error")
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

func initEnv() (queue.Queue, queue.Module, queue.Module) {
	var q = queue.New("channel")
	cfg = config.InitCfg("../cmd/chain33/chain33.test.toml")
	cfg.Consensus.Minerstart = false
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())
	exec := New(cfg.Exec)
	exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	s := store.New(cfg.Store)
	s.SetQueueClient(q.Client())
	return q, chain, s
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("none"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	random := rand.New(rand.NewSource(types.Now().UnixNano()))
	tx.Nonce = random.Int63()
	tx.To = address.ExecAddress("none")
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func createTx2(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	random := rand.New(rand.NewSource(types.Now().UnixNano()))
	tx.Nonce = random.Int63()
	tx.To = to
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func createTxWithExecer(priv crypto.PrivKey, execer string) *types.Transaction {
	to, _ := genaddress()
	if execer == "coins" {
		return createTx2(priv, to, types.Coin)
	}
	tx := &types.Transaction{Execer: []byte(execer), Payload: []byte("xxxx"), Fee: 1e6, To: address.ExecAddress(execer)}
	tx.Nonce = random.Int63()
	tx.To = address.ExecAddress(execer)
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
	addrto := address.PubKeyToAddress(privto.PubKey().Bytes())
	return addrto.String(), privto
}

func genTxs(n int64) (txs []*types.Transaction) {
	_, priv := genaddress()
	to, _ := genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, createTx(priv, to, types.Coin*(n+1)))
	}
	return txs
}

func genTxs2(priv crypto.PrivKey, n int64) (txs []*types.Transaction) {
	to, _ := genaddress()
	for i := 0; i < int(n); i++ {
		txs = append(txs, createTx2(priv, to, types.Coin*(n+1)))
	}
	return txs
}

func createBlock(n int64) *types.Block {
	newblock := &types.Block{}
	newblock.Height = -1
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = genTxs(n)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func createGenesisBlock() *types.Block {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = cfg.Consensus.Genesis
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})

	newblock := &types.Block{}
	newblock.Height = 0
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = append(newblock.Txs, &tx)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func TestExecGenesisBlock(t *testing.T) {
	q, chain, s := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	block := createGenesisBlock()
	_, _, err := ExecBlock(q.Client(), zeroHash[:], block, false, true)
	if err != nil {
		t.Error(err)
	}
}
func TestTxGroup(t *testing.T) {
	q, chain, s := initEnv()
	prev := types.MinFee
	types.SetMinFee(100000)
	defer types.SetMinFee(prev)
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	block := createGenesisBlock()
	_, _, err := ExecBlock(q.Client(), zeroHash[:], block, false, true)
	if err != nil {
		t.Error(err)
		return
	}
	printAccount(t, q.Client(), block.StateHash, cfg.Consensus.Genesis)
	var txs []*types.Transaction
	addr2, priv2 := genaddress()
	addr3, priv3 := genaddress()
	addr4, _ := genaddress()
	txs = append(txs, createTx2(genkey, addr2, types.Coin))
	txs = append(txs, createTx2(priv2, addr3, types.Coin))
	txs = append(txs, createTx2(priv3, addr4, types.Coin))
	//执行三笔交易: 全部正确
	txgroup, err := types.CreateTxGroup(txs)
	if err != nil {
		t.Error(err)
		return
	}
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, priv3)
	//返回新的区块
	block = execAndCheckBlock(t, q.Client(), block, txgroup.GetTxs(), types.ExecOk)
	printAccount(t, q.Client(), block.StateHash, cfg.Consensus.Genesis)
	printAccount(t, q.Client(), block.StateHash, addr2)
	printAccount(t, q.Client(), block.StateHash, addr3)
	printAccount(t, q.Client(), block.StateHash, addr4)
	//执行三笔交易：第一比错误
	txs = nil
	txs = append(txs, createTx2(priv2, addr3, 2*types.Coin))
	txs = append(txs, createTx2(genkey, addr4, types.Coin))
	txs = append(txs, createTx2(genkey, addr4, types.Coin))

	txgroup, err = types.CreateTxGroup(txs)
	if err != nil {
		t.Error(err)
		return
	}
	//重新签名
	txgroup.SignN(0, types.SECP256K1, priv2)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, genkey)

	block = execAndCheckBlock(t, q.Client(), block, txgroup.GetTxs(), types.ExecErr)
	//执行三笔交易：第二比错误
	txs = nil
	txs = append(txs, createTx2(genkey, addr2, types.Coin))
	txs = append(txs, createTx2(priv2, addr4, 2*types.Coin))
	txs = append(txs, createTx2(genkey, addr4, types.Coin))

	txgroup, err = types.CreateTxGroup(txs)
	if err != nil {
		t.Error(err)
		return
	}
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, genkey)

	block = execAndCheckBlock(t, q.Client(), block, txgroup.GetTxs(), types.ExecPack)
	//执行三笔交易: 第三比错误
	txs = nil
	txs = append(txs, createTx2(genkey, addr2, types.Coin))
	txs = append(txs, createTx2(genkey, addr4, types.Coin))
	txs = append(txs, createTx2(priv2, addr4, 10*types.Coin))

	txgroup, err = types.CreateTxGroup(txs)
	if err != nil {
		t.Error(err)
		return
	}
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, priv2)

	block = execAndCheckBlock(t, q.Client(), block, txgroup.GetTxs(), types.ExecPack)
}

func TestExecAllow(t *testing.T) {
	q, chain, s := initEnv()
	prev := types.MinFee
	types.SetMinFee(100000)
	defer types.SetMinFee(prev)

	defer chain.Close()
	defer s.Close()
	defer q.Close()
	block := createGenesisBlock()
	_, _, err := ExecBlock(q.Client(), zeroHash[:], block, false, true)
	if err != nil {
		t.Error(err)
		return
	}
	types.SetTitle("bityuan")
	defer types.SetTitle("local")
	tx1 := createTxWithExecer(genkey, "user.evm")     //allow
	tx2 := createTxWithExecer(genkey, "coins")        //allow
	tx3 := createTxWithExecer(genkey, "evm")          //not allow
	tx4 := createTxWithExecer(genkey, "user.evm.xxx") //not allow
	printAccount(t, q.Client(), block.StateHash, cfg.Consensus.Genesis)
	txs := []*types.Transaction{tx1, tx2, tx3, tx4}
	block = execAndCheckBlockCB(t, q.Client(), block, txs, func(index int, receipt *types.ReceiptData) error {
		if index == 0 && receipt.GetTy() != 1 {
			return errors.New("user.evm is allow")
		}
		if index == 1 && receipt.GetTy() != 2 {
			return errors.New("coins exec ok")
		}
		if index == 2 && receipt != nil {
			return errors.New("evm is not allow in bityuan")
		}
		if index == 3 && receipt != nil {
			return errors.New("user.evm.xxx is not allow in bityuan")
		}
		return nil
	})
}

func execAndCheckBlock(t *testing.T, qclient queue.Client,
	block *types.Block, txs []*types.Transaction, result int) *types.Block {
	block2 := createNewBlock(t, block, txs)
	detail, deltx, err := ExecBlock(qclient, block.StateHash, block2, false, true)
	if err != nil {
		t.Error(err)
		return nil
	}
	if result == 0 && len(deltx) != len(txs) {
		t.Error("must all failed")
		return nil
	}
	if result > 0 && len(deltx) != 0 {
		t.Error("del tx is zero")
		return nil
	}
	for i := 0; i < len(detail.Block.Txs); i++ {
		if detail.Receipts[i].GetTy() != int32(result) {
			t.Errorf("exec expect all is %d, but now %d, index %d", result, detail.Receipts[i].GetTy(), i)
		}
	}
	return detail.Block
}

func execAndCheckBlockCB(t *testing.T, qclient queue.Client,
	block *types.Block, txs []*types.Transaction, cb func(int, *types.ReceiptData) error) *types.Block {
	block2 := createNewBlock(t, block, txs)
	detail, deltx, err := ExecBlock(qclient, block.StateHash, block2, false, true)
	if err != nil {
		t.Error(err)
		return nil
	}

	var getIndex = func(hash []byte, txlist []*types.Transaction) int {
		for i := 0; i < len(txlist); i++ {
			if bytes.Equal(hash, txlist[i].Hash()) {
				return i
			}
		}
		return -1
	}
	for i := 0; i < len(txs); i++ {
		if getIndex(txs[i].Hash(), deltx) >= 0 {
			if err := cb(i, nil); err != nil {
				t.Error(err, "index", i)
				return nil
			}
		} else if getIndex(txs[i].Hash(), detail.Block.Txs) >= 0 {
			if err := cb(i, detail.Receipts[i]); err != nil {
				t.Error(err, "index", i)
				return nil
			}
		} else {
			panic("never happen")
		}
	}
	return detail.Block
}

func TestExecBlock2(t *testing.T) {
	q, chain, s := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()

	block := createGenesisBlock()
	detail, _, err := ExecBlock(q.Client(), zeroHash[:], block, false, true)
	if err != nil {
		t.Error(err)
	}
	printAccount(t, q.Client(), detail.Block.StateHash, cfg.Consensus.Genesis)
	txs := genTxs2(genkey, 2)

	block2 := createNewBlock(t, block, txs)
	detail, _, err = ExecBlock(q.Client(), block.StateHash, block2, false, true)
	if err != nil {
		t.Error(err)
		return
	}
	for _, Receipt := range detail.Receipts {
		if Receipt.GetTy() != 2 {
			t.Errorf("exec expect true, but now false")
		}
	}

	N := 5000
	done := make(chan struct{}, N)
	for i := 0; i < N; i++ {
		go func() {
			txs := genTxs2(genkey, 2)
			block3 := createNewBlock(t, block2, txs)
			detail, _, err := ExecBlock(q.Client(), block2.StateHash, block3, false, true)
			if err != nil {
				t.Error(err)
				return
			}
			for _, Receipt := range detail.Receipts {
				if Receipt.GetTy() != 2 {
					t.Errorf("exec expect true, but now false")
				}
			}
			done <- struct{}{}
		}()
	}
	for n := 0; n < N; n++ {
		<-done
	}
}

func printAccount(t *testing.T, qclient queue.Client, stateHash []byte, addr string) {
	statedb := NewStateDB(qclient, stateHash, nil)
	acc := account.NewCoinsAccount()
	acc.SetDB(statedb)
	t.Log(acc.LoadAccount(addr))
}

func jsonPrint(t *testing.T, input interface{}) {
	data, err := json.MarshalIndent(input, "", "\t")
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(string(data))
}

func createNewBlock(t *testing.T, parent *types.Block, txs []*types.Transaction) *types.Block {
	newblock := &types.Block{}
	newblock.Height = parent.Height + 1
	newblock.BlockTime = parent.BlockTime + 1
	newblock.ParentHash = parent.Hash()
	newblock.Txs = append(newblock.Txs, txs...)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func TestExecBlock(t *testing.T) {
	q, chain, s := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	block := createBlock(10)
	ExecBlock(q.Client(), zeroHash[:], block, false, true)
}

//gen 1万币需要 2s，主要是签名的花费
func BenchmarkGenRandBlock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		createBlock(10000)
	}
}

func BenchmarkExecBlock(b *testing.B) {
	q, chain, s := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	block := createBlock(10000)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		ExecBlock(q.Client(), zeroHash[:], block, false, true)
	}
}

func TestLoadDriver(t *testing.T) {
	d, err := drivers.LoadDriver("none", 0)
	if err != nil {
		t.Error(err)
	}

	if d.GetName() != "none" {
		t.Error(d.GetName())
	}
}

func TestKeyAllow(t *testing.T) {
	key := []byte("mavl-coins-bty-exec-1wvmD6RNHzwhY4eN75WnM6JcaAvNQ4nHx:19xXg1WHzti5hzBRTUphkM8YmuX6jJkoAA")
	exec := []byte("retrieve")
	if !isAllowExec(key, exec, address.ExecAddress("retrieve"), int64(1)) {
		t.Error("retrieve can modify exec")
	}
}

func ExecBlock(client queue.Client, prevStateRoot []byte, block *types.Block, errReturn bool, sync bool) (*types.BlockDetail, []*types.Transaction, error) {
	//发送执行交易给execs模块
	//通过consensus module 再次检查
	ulog := elog
	ulog.Debug("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
	beg := types.Now()
	defer func() {
		ulog.Info("ExecBlock", "height", block.Height, "ntx", len(block.Txs), "writebatchsync", sync, "cost", types.Since(beg))
	}()
	if errReturn && block.Height > 0 && !block.CheckSign() {
		//block的来源不是自己的mempool，而是别人的区块
		return nil, nil, types.ErrSign
	}
	//tx交易去重处理, 这个地方要查询数据库，需要一个更快的办法
	cacheTxs := types.TxsToCache(block.Txs)
	oldtxscount := len(cacheTxs)
	cacheTxs = util.CheckTxDup(client, cacheTxs, block.Height)
	newtxscount := len(cacheTxs)
	if oldtxscount != newtxscount && errReturn {
		return nil, nil, types.ErrTxDup
	}
	ulog.Debug("ExecBlock", "prevtx", oldtxscount, "newtx", newtxscount)
	block.TxHash = merkle.CalcMerkleRootCache(cacheTxs)
	block.Txs = types.CacheToTxs(cacheTxs)

	receipts := util.ExecTx(client, prevStateRoot, block)
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
		calcHash = util.ExecKVMemSet(client, prevStateRoot, kvset, sync)
	}
	if errReturn && !bytes.Equal(block.StateHash, calcHash) {
		util.ExecKVSetRollback(client, calcHash)
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
	//save to db
	if kvset != nil {
		util.ExecKVSetCommit(client, block.StateHash)
	}
	return &detail, deltx, nil
}
