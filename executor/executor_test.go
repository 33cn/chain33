// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"encoding/hex"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/33cn/chain33/client"
	"github.com/33cn/chain33/client/api"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/store"
	_ "github.com/33cn/chain33/system"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func initEnv(cfgstring string) (*Executor, queue.Queue) {
	cfg := types.NewChain33Config(cfgstring)
	q := queue.New("channel")
	q.SetConfig(cfg)
	exec := New(cfg)
	exec.client = q.Client()
	exec.qclient, _ = client.New(exec.client, nil)
	return exec, q
}

func TestIsModule(t *testing.T) {
	var qmodule queue.Module = &Executor{}
	assert.NotNil(t, qmodule)
}

func TestExecutorGetTxGroup(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	execInit(nil)
	var txs []*types.Transaction
	addr2, priv2 := util.Genaddress()
	addr3, priv3 := util.Genaddress()
	addr4, _ := util.Genaddress()
	genkey := util.TestPrivkeyList[0]
	txs = append(txs, util.CreateCoinsTx(cfg, genkey, addr2, types.Coin))
	txs = append(txs, util.CreateCoinsTx(cfg, priv2, addr3, types.Coin))
	txs = append(txs, util.CreateCoinsTx(cfg, priv3, addr4, types.Coin))
	//执行三笔交易: 全部正确
	txgroup, err := types.CreateTxGroup(txs, cfg.GetMinTxFeeRate())
	if err != nil {
		t.Error(err)
		return
	}
	//重新签名
	txgroup.SignN(0, types.SECP256K1, genkey)
	txgroup.SignN(1, types.SECP256K1, priv2)
	txgroup.SignN(2, types.SECP256K1, priv3)
	txs = txgroup.GetTxs()
	ctx := &executorCtx{
		stateHash:  nil,
		height:     1,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	execute := newExecutor(ctx, exec, nil, txs, nil)
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
	execute = newExecutor(ctx, exec, nil, txs, nil)
	e = execute.loadDriver(txs[0], 0)
	execute.setEnv(e)
	_, err = e.GetTxGroup(len(txs) - 1)
	assert.Equal(t, err, types.ErrTxGroupFormat)
}

//gen 1万币需要 2s，主要是签名的花费
func BenchmarkGenRandBlock(b *testing.B) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	_, key := util.Genaddress()
	for i := 0; i < b.N; i++ {
		util.CreateNoneBlock(cfg, key, 10000)
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
	exect, _ := initEnv(types.GetDefaultCfgstring())
	execInit(nil)
	key := []byte("mavl-coins-bty-exec-1wvmD6RNHzwhY4eN75WnM6JcaAvNQ4nHx:19xXg1WHzti5hzBRTUphkM8YmuX6jJkoAA")
	exec := []byte("retrieve")
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	ctx := &executorCtx{
		stateHash:  nil,
		height:     1,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	execute := newExecutor(ctx, exect, nil, nil, nil)
	if !isAllowKeyWrite(execute, key, exec, &tx12, 0) {
		t.Error("retrieve can modify exec")
	}
}

func TestKeyAllow_evm(t *testing.T) {
	exect, _ := initEnv(types.GetDefaultCfgstring())
	execInit(nil)
	key := []byte("mavl-coins-bty-exec-1GacM93StrZveMrPjXDoz5TxajKa9LM5HG:19EJVYexvSn1kZ6MWiKcW14daXsPpdVDuF")
	exec := []byte("user.evm.0xc79c9113a71c0a4244e20f0780e7c13552f40ee30b05998a38edb08fe617aaa5")
	tx1 := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	tx11, _ := hex.DecodeString(tx1)
	var tx12 types.Transaction
	types.Decode(tx11, &tx12)
	tx12.Execer = exec
	ctx := &executorCtx{
		stateHash:  nil,
		height:     1,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	execute := newExecutor(ctx, exect, nil, nil, nil)
	if !isAllowKeyWrite(execute, key, exec, &tx12, 0) {
		t.Error("user.evm.hash can modify exec")
	}
	//assert.Nil(t, t)
}

func TestKeyLocalAllow(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	err := isAllowLocalKey(cfg, []byte("token"), []byte("LODB-token-"))
	assert.Equal(t, err, types.ErrLocalKeyLen)
	err = isAllowLocalKey(cfg, []byte("token"), []byte("LODB_token-a"))
	assert.Equal(t, err, types.ErrLocalPrefix)
	err = isAllowLocalKey(cfg, []byte("token"), []byte("LODB-token-a"))
	assert.Nil(t, err)
	err = isAllowLocalKey(cfg, []byte(""), []byte("LODB--a"))
	assert.Equal(t, err, types.ErrLocalPrefix)
	err = isAllowLocalKey(cfg, []byte("exec"), []byte("LODB-execaa"))
	assert.Equal(t, err, types.ErrLocalPrefix)
	err = isAllowLocalKey(cfg, []byte("exec"), []byte("-exec------aa"))
	assert.Equal(t, err, types.ErrLocalPrefix)
	err = isAllowLocalKey(cfg, []byte("paracross"), []byte("LODB-user.p.para.paracross-xxxx"))
	assert.Equal(t, err, types.ErrLocalPrefix)
	err = isAllowLocalKey(cfg, []byte("user.p.para.paracross"), []byte("LODB-user.p.para.paracross-xxxx"))
	assert.Nil(t, err)
	err = isAllowLocalKey(cfg, []byte("user.p.para.user.wasm.abc"), []byte("LODB-user.p.para.user.wasm.abc-xxxx"))
	assert.Nil(t, err)
	err = isAllowLocalKey(cfg, []byte("user.p.para.paracross"), []byte("LODB-paracross-xxxx"))
	assert.Nil(t, err)
}

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte("demo"), []byte("demof"))
}

var testRegOnce sync.Once

func Register(cfg *types.Chain33Config) {
	testRegOnce.Do(func() {
		drivers.Register(cfg, "demo", newdemoApp, 1)
		drivers.Register(cfg, "demof", newdemofApp, 1)
	})
}

//ErrEnvAPI 测试
type demoApp struct {
	*drivers.DriverBase
}

func newdemoApp() drivers.Driver {
	demo := &demoApp{DriverBase: &drivers.DriverBase{}}
	demo.SetChild(demo)
	return demo
}

func (demo *demoApp) GetDriverName() string {
	return "demo"
}

func (demo *demoApp) Exec(tx *types.Transaction, index int) (receipt *types.Receipt, err error) {
	return nil, queue.ErrQueueTimeout
}

func (demo *demoApp) Upgrade() (*types.LocalDBSet, error) {
	db := demo.GetLocalDB()
	db.Set([]byte("LODB-demo-a"), []byte("t1"))
	db.Set([]byte("LODB-demo-b"), []byte("t2"))
	var kvset types.LocalDBSet
	kvset.KV = []*types.KeyValue{
		{Key: []byte("LODB-demo-a"), Value: []byte("t1")},
		{Key: []byte("LODB-demo-b"), Value: []byte("t2")},
	}
	return &kvset, nil
}

type demofApp struct {
	*drivers.DriverBase
}

func newdemofApp() drivers.Driver {
	demo := &demofApp{DriverBase: &drivers.DriverBase{}}
	demo.SetChild(demo)
	return demo
}

func (demo *demofApp) GetDriverName() string {
	return "demof"
}

func (demo *demofApp) Exec(tx *types.Transaction, index int) (receipt *types.Receipt, err error) {
	return nil, queue.ErrQueueTimeout
}

func (demo *demofApp) Upgrade() (kvset *types.LocalDBSet, err error) {
	return nil, types.ErrInvalidParam
}

func TestExecutorErrAPIEnv(t *testing.T) {
	exec, q := initEnv(types.GetDefaultCfgstring())
	exec.disableLocal = true
	cfg := exec.client.GetConfig()
	cfg.SetMinFee(0)
	Register(cfg)
	execInit(cfg)

	store := store.New(cfg)
	store.SetQueueClient(q.Client())
	defer store.Close()

	var txs []*types.Transaction
	genkey := util.TestPrivkeyList[0]
	txs = append(txs, util.CreateTxWithExecer(cfg, genkey, "demo"))
	txlist := &types.ExecTxList{
		StateHash:  nil,
		Height:     1,
		BlockTime:  time.Now().Unix(),
		Difficulty: 1,
		MainHash:   nil,
		MainHeight: 1,
		ParentHash: nil,
		Txs:        txs,
	}
	msg := queue.NewMessage(0, "", 1, txlist)
	exec.procExecTxList(msg)
	_, err := exec.client.WaitTimeout(msg, 100*time.Second)
	fmt.Println(err)
	assert.Equal(t, true, api.IsAPIEnvError(err))
}
func TestCheckTx(t *testing.T) {
	exec, q := initEnv(types.ReadFile("../cmd/chain33/chain33.test.toml"))
	cfg := exec.client.GetConfig()

	store := store.New(cfg)
	store.SetQueueClient(q.Client())
	defer store.Close()

	addr, priv := util.Genaddress()

	tx := util.CreateCoinsTx(cfg, priv, addr, types.Coin)
	tx.Execer = []byte("user.xxx")
	tx.To = address.ExecAddress("user.xxx")
	tx.Fee = 2 * types.Coin
	tx.Sign(types.SECP256K1, priv)

	var txs []*types.Transaction
	txs = append(txs, tx)
	ctx := &executorCtx{
		stateHash:  nil,
		height:     0,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
		mainHash:   nil,
		parentHash: nil,
	}
	execute := newExecutor(ctx, exec, nil, txs, nil)
	err := execute.execCheckTx(tx, 0)
	assert.Equal(t, err, types.ErrNoBalance)
}

func TestExecutorUpgradeMsg(t *testing.T) {
	exec, q := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	cfg.SetMinFee(0)
	Register(cfg)
	execInit(cfg)

	exec.SetQueueClient(q.Client())
	client := q.Client()
	msg := client.NewMessage("execs", types.EventUpgrade, nil)
	err := client.Send(msg, true)
	assert.Nil(t, err)
}

func TestExecutorPluginUpgrade(t *testing.T) {
	exec, q := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	cfg.SetMinFee(0)
	Register(cfg)
	execInit(cfg)

	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			if msg.Ty == types.EventLocalNew {
				msg.Reply(client.NewMessage("", types.EventHeader, &types.Int64{Data: 100}))
			} else {
				msg.Reply(client.NewMessage("", types.EventHeader, &types.Header{Height: 100}))
			}
		}
	}()

	kvset, err := exec.upgradePlugin("demo")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(kvset.GetKV()))
	kvset, err = exec.upgradePlugin("demof")
	assert.NotNil(t, err)
	assert.Nil(t, kvset)
}
