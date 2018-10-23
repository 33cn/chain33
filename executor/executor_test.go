package executor

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"testing"
	"time"

	"encoding/hex"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/config"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/limits"
	"gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/p2p"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/store"
	drivers "gitlab.33.cn/chain33/chain33/system/dapp"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"

	"net/http"
	_ "net/http/pprof"

	_ "gitlab.33.cn/chain33/chain33/plugin"
	_ "gitlab.33.cn/chain33/chain33/system"
)

var random *rand.Rand
var zeroHash [32]byte
var cfg *types.Config
var sub *types.ConfigSubModule
var genkey crypto.PrivKey

func init() {
	go func() {
		http.ListenAndServe("localhost:6060", nil)
	}()
	types.SetTitle("local")
	types.SetForkToOne()
	types.SetTestNet(true)
	err := limits.SetLimits()
	if err != nil {
		panic(err)
	}
	pluginmgr.InitExec(nil)
	random = rand.New(rand.NewSource(types.Now().UnixNano()))
	genkey = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")
	log.SetLogLevel("error")
}

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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

func initEnv() (queue.Queue, queue.Module, queue.Module, queue.Module) {
	var q = queue.New("channel")
	cfg, sub = config.InitCfg("../cmd/chain33/chain33.test.toml")
	cfg.Consensus.Minerstart = false
	chain := blockchain.New(cfg.BlockChain)
	chain.SetQueueClient(q.Client())
	exec := New(cfg.Exec, sub.Exec)
	exec.SetQueueClient(q.Client())
	types.SetMinFee(0)
	s := store.New(cfg.Store, sub.Store)
	s.SetQueueClient(q.Client())
	network := p2p.New(cfg.P2P)
	network.SetQueueClient(q.Client())

	return q, chain, s, network
}

func createTx(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: amount}}
	transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
	tx := &types.Transaction{Execer: []byte("none"), Payload: types.Encode(transfer), Fee: 1e6, To: to}
	random := rand.New(rand.NewSource(types.Now().UnixNano()))
	tx.Nonce = random.Int63()
	tx.To = address.ExecAddress("none")
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func createTx2(priv crypto.PrivKey, to string, amount int64) *types.Transaction {
	v := &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: amount}}
	transfer := &cty.CoinsAction{Value: v, Ty: cty.CoinsActionTransfer}
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
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
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
	g := &cty.CoinsAction_Genesis{}
	g.Genesis = &types.AssetsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&cty.CoinsAction{Value: g, Ty: cty.CoinsActionGenesis})

	newblock := &types.Block{}
	newblock.Height = 0
	newblock.BlockTime = types.Now().Unix()
	newblock.ParentHash = zeroHash[:]
	newblock.Txs = append(newblock.Txs, &tx)
	newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
	return newblock
}

func TestExecGenesisBlock(t *testing.T) {
	q, chain, s, p2p := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer p2p.Close()
	block := createGenesisBlock()
	_, _, err := ExecBlock(q.Client(), zeroHash[:], block, false, true)
	if err != nil {
		t.Error(err)
	}
}

func TestExecutorGetTxGroup(t *testing.T) {
	exec := &Executor{}
	execInit(nil)
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
	txs = txgroup.GetTxs()
	execute := newExecutor(nil, exec, 1, time.Now().Unix(), 1, txs, nil)
	e := execute.loadDriver(txs[0], 0)
	execute.setEnv(e)
	txs2 := e.GetTxs()
	assert.Equal(t, txs2, txgroup.GetTxs())
	for i := 0; i < len(txs); i++ {
		txg, err := e.GetTxGroup(i)
		assert.Nil(t, err)
		assert.Equal(t, txg, txgroup.GetTxs())
	}
	_, err = e.GetTxGroup(len(txs))
	assert.Equal(t, err, types.ErrTxGroupIndex)

	//err tx group list
	txs[0].Header = nil
	execute = newExecutor(nil, exec, 1, time.Now().Unix(), 1, txs, nil)
	e = execute.loadDriver(txs[0], 0)
	execute.setEnv(e)
	_, err = e.GetTxGroup(len(txs) - 1)
	assert.Equal(t, err, types.ErrTxGroupFormat)
}

func TestTxGroup(t *testing.T) {
	q, chain, s, p2p := initEnv()
	prev := types.MinFee
	types.SetMinFee(100000)
	defer types.SetMinFee(prev)
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer p2p.Close()
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

	//执行三笔交易：其中有一笔是user.xxx的执行器
	txs = nil
	txs = append(txs, createTx2(genkey, addr2, types.Coin))
	txs = append(txs, createTx2(genkey, addr4, types.Coin))
	txs = append(txs, createTx2(priv2, addr4, 10*types.Coin))
	txs[2].Execer = []byte("user.xxx")
	txs[2].To = address.ExecAddress("user.xxx")
	txgroup, err = types.CreateTxGroup(txs)
	if err != nil {
		t.Error(err)
		return
	}
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, genkey)
	txgroup.SignN(2, types.SECP256K1, priv2)

	block = execAndCheckBlock2(t, q.Client(), block, txgroup.GetTxs(), []int{2, 2, 1})
}

func TestExecAllow(t *testing.T) {
	q, chain, s, p2p := initEnv()
	prev := types.MinFee
	types.SetMinFee(100000)
	defer types.SetMinFee(prev)

	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer p2p.Close()
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

func execAndCheckBlock2(t *testing.T, qclient queue.Client,
	block *types.Block, txs []*types.Transaction, result []int) *types.Block {
	block2 := createNewBlock(t, block, txs)
	detail, _, err := ExecBlock(qclient, block.StateHash, block2, false, true)
	if err != nil {
		t.Error(err)
		return nil
	}
	for i := 0; i < len(detail.Block.Txs); i++ {
		if detail.Receipts[i].GetTy() != int32(result[i]) {
			t.Errorf("exec expect all is %d, but now %d, index %d", result[i], detail.Receipts[i].GetTy(), i)
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
	for _, v := range deltx {
		s, err := types.PBToJson(v)
		if err != nil {
			t.Error(err)
			return nil
		}
		println(s)
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
				t.Error(err, "i", i)
				return nil
			}
		} else if index := getIndex(txs[i].Hash(), detail.Block.Txs); index >= 0 {
			if err := cb(i, detail.Receipts[index]); err != nil {
				t.Error(err, "i", i, "index", index)
				return nil
			}
		} else {
			panic("never happen")
		}
	}
	return detail.Block
}

func TestExecBlock2(t *testing.T) {
	q, chain, s, p2p := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer p2p.Close()
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

	N := 1000
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
	statedb := NewStateDB(qclient, stateHash, nil, nil)
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
	q, chain, s, p2p := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer p2p.Close()
	block := createBlock(10)
	ExecBlock(q.Client(), zeroHash[:], block, false, true)
}

//gen 1万币需要 2s，主要是签名的花费
func BenchmarkGenRandBlock(b *testing.B) {
	for i := 0; i < b.N; i++ {
		createBlock(10000)
	}
}

//区块执行性能更好的一个测试
//1. 先生成 10万个账户，每个账户转1000个币
//2. 每个区块随机取1万比交易，然后执行
//3. 执行1000个块，看性能曲线的变化
//4. 排除掉网络掉影响
//5. 先对leveldb 做一个性能的测试

//区块执行新能测试
func BenchmarkExecBlock(b *testing.B) {
	q, chain, s, p2p := initEnv()
	defer chain.Close()
	defer s.Close()
	defer q.Close()
	defer p2p.Close()
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
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if !isAllowExec(key, exec, &tx12, int64(1)) {
		t.Error("retrieve can modify exec")
	}
}

func TestKeyAllow_evm(t *testing.T) {
	key := []byte("mavl-coins-bty-exec-1GacM93StrZveMrPjXDoz5TxajKa9LM5HG:19EJVYexvSn1kZ6MWiKcW14daXsPpdVDuF")
	exec := []byte("user.evm.0xc79c9113a71c0a4244e20f0780e7c13552f40ee30b05998a38edb08fe617aaa5")
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if !isAllowExec(key, exec, &tx12, int64(1)) {
		t.Error("user.evm.hash can modify exec")
	}
}

func TestKeyAllow_ticket(t *testing.T) {
	key := []byte("mavl-coins-bty-exec-16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp")
	exec := []byte("ticket")
	tx1 := "0a067469636b657412c701501022c20108dcaed4f1011080a4a7da061a70314556474572784d52343565577532386d6f4151616e6b34413864516635623639383a3078356161303431643363623561356230396131333336626536373539356638366461336233616564386531653733373139346561353135313562653336363933333a3030303030303030303022423078336461373533326364373839613330623037633538343564336537383433613731356630393961616566386533646161376134383765613135383135336331631a6e08011221025a317f60e6962b7ce9836a83b775373b614b290bee595f8aecee5499791831c21a473045022100850bb15cdcdaf465af7ad1ffcbc1fd6a86942a1ddec1dc112164f37297e06d2d02204aca9686fd169462be955cef1914a225726280739770ab1c0d29eb953e54c6b620a08d0630e3faecf8ead9f9e1483a22313668747663424e53454137665a6841644c4a706844775152514a61487079485470"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	if !isAllowExec(key, exec, &tx12, int64(1)) {
		t.Error("ticket can modify exec")
	}
}

func TestKeyAllow_paracross(t *testing.T) {
	key := []byte("mavl-coins-bty-exec-1HPkPopVe3ERfvaAgedDtJQ792taZFEHCe:19xXg1WHzti5hzBRTUphkM8YmuX6jJkoAA")
	exec := []byte("paracross")
	tx1 := "0a15757365722e702e746573742e7061726163726f7373124310904e223e1080c2d72f1a1374657374206173736574207472616e736665722222314a524e6a64457170344c4a356671796355426d396179434b536565736b674d4b5220a08d0630f7cba7ec9e8f9bac163a2231367a734d68376d764e444b50473645394e5672506877367a4c3933675773547052"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = []byte("user.p.para.paracross")
	if !isAllowExec(key, exec, &tx12, int64(1)) {
		t.Error("paracross can modify exec")
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
	var err error
	cacheTxs, err = util.CheckTxDup(client, cacheTxs, block.Height)
	if err != nil {
		return nil, nil, err
	}
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
	//if kvset == nil {
	//	calcHash = prevStateRoot
	//} else {
	calcHash = util.ExecKVMemSet(client, prevStateRoot, block.Height, kvset, sync)
	//}
	if errReturn && !bytes.Equal(block.StateHash, calcHash) {
		util.ExecKVSetRollback(client, calcHash)
		if len(rdata) > 0 {
			for i, rd := range rdata {
				rd.OutputReceiptDetails(block.Txs[i].Execer, ulog)
			}
		}
		return nil, nil, types.ErrCheckStateHash
	}
	block.StateHash = calcHash
	detail.Block = block
	detail.Receipts = rdata
	//save to db
	//if kvset != nil {
	util.ExecKVSetCommit(client, block.StateHash)
	//}
	return &detail, deltx, nil
}
