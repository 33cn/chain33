package dapp

import (
	"testing"
	"time"

	"sync"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/rpc/grpcclient"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var runonce sync.Once

type demoApp struct {
	*DriverBase
}

func newdemoApp() Driver {
	demo := &demoApp{DriverBase: &DriverBase{}}
	demo.SetChild(demo)
	return demo
}

func (demo *demoApp) GetDriverName() string {
	return "demo"
}

type noneApp struct {
	*DriverBase
}

func newnoneApp() Driver {
	none := &noneApp{DriverBase: &DriverBase{}}
	none.SetChild(none)
	return none
}

func (none *noneApp) GetDriverName() string {
	return "none"
}

func Init(cfg *types.Chain33Config) {
	runonce.Do(func() {
		Register(cfg, "none", newnoneApp, 0)
		Register(cfg, "demo", newdemoApp, 1)
	})
}

func TestReigister(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	Init(cfg)
	api := &mocks.QueueProtocolAPI{}
	api.On("GetConfig", mock.Anything).Return(cfg)

	_, err := LoadDriver("demo", 0)
	assert.Equal(t, err, types.ErrUnknowDriver)
	_, err = LoadDriver("demo", 1)
	assert.Equal(t, err, nil)

	tx := &types.Transaction{Execer: []byte("demo")}
	driver := LoadDriverAllow(api, tx, 0, 0)
	assert.Equal(t, "none", driver.GetDriverName())
	driver = LoadDriverAllow(api, tx, 0, 1)
	assert.Equal(t, "demo", driver.GetDriverName())

	cfg.SetTitleOnlyForTest("user.p.hello.")
	tx = &types.Transaction{Execer: []byte("demo")}
	driver = LoadDriverAllow(api, tx, 0, 0)
	assert.Equal(t, "none", driver.GetDriverName())
	driver = LoadDriverAllow(api, tx, 0, 1)
	assert.Equal(t, "demo", driver.GetDriverName())

	tx.Execer = []byte("user.p.hello.demo")
	driver = LoadDriverAllow(api, tx, 0, 1)
	assert.Equal(t, "demo", driver.GetDriverName())

	tx.Execer = []byte("user.p.hello2.demo")
	driver = LoadDriverAllow(api, tx, 0, 1)
	assert.Equal(t, "none", driver.GetDriverName())
}

func TestDriverAPI(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	Init(cfg)
	api := &mocks.QueueProtocolAPI{}
	api.On("GetConfig", mock.Anything).Return(cfg)

	tx := &types.Transaction{Execer: []byte("demo")}
	demo := LoadDriverAllow(api, tx, 0, 1).(*demoApp)
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	demo.SetEnv(1, time.Now().Unix(), 1)
	demo.SetBlockInfo([]byte("parentHash"), []byte("mainHash"), 1)
	demo.SetLocalDB(kvdb)
	demo.SetStateDB(kvdb)
	demo.SetAPI(api)
	gcli, err := grpcclient.NewMainChainClient(cfg, "")
	assert.Nil(t, err)
	demo.SetExecutorAPI(api, gcli)
	assert.NotNil(t, demo.GetAPI())
	assert.NotNil(t, demo.GetExecutorAPI())
	cfg.SetTitleOnlyForTest("chain33")
	assert.Equal(t, "parentHash", string(demo.GetParentHash()))
	assert.Equal(t, "parentHash", string(demo.GetLastHash()))
	cfg.SetTitleOnlyForTest("user.p.wzw.")
	assert.Equal(t, "parentHash", string(demo.GetParentHash()))
	assert.Equal(t, "mainHash", string(demo.GetLastHash()))
	assert.Equal(t, int64(1), demo.GetMainHeight())
	assert.Equal(t, true, IsDriverAddress(ExecAddress("none"), 0))
	assert.Equal(t, false, IsDriverAddress(ExecAddress("demo"), 0))
	assert.Equal(t, true, IsDriverAddress(ExecAddress("demo"), 1))
}

func TestExecAddress(t *testing.T) {
	assert.Equal(t, "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", ExecAddress("ticket"))
}

func TestAllow(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	Init(cfg)
	api := &mocks.QueueProtocolAPI{}
	api.On("GetConfig", mock.Anything).Return(cfg)

	tx := &types.Transaction{Execer: []byte("demo")}
	demo := LoadDriverAllow(api, tx, 0, 1).(*demoApp)
	assert.Equal(t, true, demo.AllowIsSame([]byte("demo")))
	cfg.SetTitleOnlyForTest("user.p.wzw.")
	assert.Equal(t, true, demo.AllowIsSame([]byte("user.p.wzw.demo")))
	assert.Equal(t, false, demo.AllowIsSame([]byte("user.p.wzw2.demo")))
	assert.Equal(t, false, demo.AllowIsUserDot1([]byte("user.p.wzw.demo")))
	assert.Equal(t, true, demo.AllowIsUserDot1([]byte("user.demo")))
	assert.Equal(t, true, demo.AllowIsUserDot1([]byte("user.p.wzw.user.demo")))
	assert.Equal(t, true, demo.AllowIsUserDot2([]byte("user.p.wzw.user.demo.xxxx")))
	assert.Equal(t, true, demo.AllowIsUserDot2([]byte("user.demo.xxxx")))
	assert.Equal(t, nil, demo.Allow(tx, 0))
	tx = &types.Transaction{Execer: []byte("demo2")}
	assert.Equal(t, types.ErrNotAllow, demo.Allow(tx, 0))
	assert.Equal(t, false, demo.IsFriend(nil, nil, nil))
}

func TestDriverBase(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	api := &mocks.QueueProtocolAPI{}
	api.On("GetConfig", mock.Anything).Return(cfg)
	Init(cfg)
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	demo := newdemoApp().(*demoApp)
	demo.SetAPI(api)
	demo.SetExecutorType(nil)
	assert.Nil(t, demo.GetPayloadValue())
	assert.Nil(t, demo.GetExecutorType())
	assert.True(t, demo.ExecutorOrder() == 0)
	assert.Nil(t, demo.GetFuncMap())
	demo.SetIsFree(false)
	assert.False(t, demo.IsFree())

	tx := &types.Transaction{Execer: []byte("demo"), To: ExecAddress("demo"), GroupCount: 1}
	t.Log("addr:", ExecAddress("demo"))
	_, err := demo.ExecLocal(tx, nil, 0)
	assert.NoError(t, err)
	_, err = demo.ExecDelLocal(tx, nil, 0)
	assert.NoError(t, err)
	_, err = demo.Exec(tx, 0)
	assert.NoError(t, err)
	err = demo.CheckTx(tx, 0)
	assert.NoError(t, err)

	txs := []*types.Transaction{tx}
	demo.SetTxs(txs)
	assert.Equal(t, txs, demo.GetTxs())
	_, err = demo.GetTxGroup(0)
	assert.Equal(t, types.ErrTxGroupFormat, err)

	demo.SetReceipt(nil)
	assert.Nil(t, demo.GetReceipt())
	demo.SetLocalDB(nil)
	assert.Nil(t, demo.GetLocalDB())
	assert.Nil(t, demo.GetStateDB())
	assert.True(t, demo.GetHeight() == 0)
	assert.True(t, demo.GetBlockTime() == 0)
	assert.True(t, demo.GetDifficulty() == 0)
	assert.Equal(t, "demo", demo.GetName())
	assert.Equal(t, "demo", demo.GetCurrentExecName())

	name := demo.GetActionName(tx)
	assert.Equal(t, "unknown", name)
	assert.True(t, demo.CheckSignatureData(tx, 0))
	assert.NotNil(t, demo.GetCoinsAccount())
	assert.False(t, demo.CheckReceiptExecOk())

	err = CheckAddress(cfg, "1HUiTRFvp6HvW6eacgV9EoBSgroRDiUsMs", 0)
	assert.NoError(t, err)

	demo.SetLocalDB(kvdb)
	execer := "user.p.guodun.demo"
	kvs := []*types.KeyValue{
		{
			Key:   []byte("hello"),
			Value: []byte("world"),
		},
	}
	newkvs := demo.AddRollbackKV(tx, []byte(execer), kvs)
	assert.Equal(t, 2, len(newkvs))
	assert.Equal(t, string(newkvs[0].Key), "hello")
	assert.Equal(t, string(newkvs[0].Value), "world")
	assert.Equal(t, string(newkvs[1].Key), string(append([]byte("LODB-demo-rollback-"), tx.Hash()...)))

	rollbackkvs := []*types.KeyValue{
		{
			Key:   []byte("hello"),
			Value: nil,
		},
	}
	data := types.Encode(&types.LocalDBSet{KV: rollbackkvs})
	assert.Equal(t, string(newkvs[1].Value), string(types.Encode(&types.ReceiptLog{Ty: types.TyLogRollback, Log: data})))

	kvdb.Set(newkvs[1].Key, newkvs[1].Value)
	newkvs, err = demo.DelRollbackKV(tx, []byte(execer))
	assert.Nil(t, err)
	assert.Equal(t, 2, len(newkvs))
	assert.Equal(t, string(newkvs[0].Key), "hello")
	assert.Equal(t, newkvs[0].Value, []byte(nil))

	assert.Equal(t, string(newkvs[1].Key), string(append([]byte("LODB-demo-rollback-"), tx.Hash()...)))
	assert.Equal(t, newkvs[1].Value, []byte(nil))
}

func TestDriverBase_Query(t *testing.T) {
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	demo := newdemoApp().(*demoApp)
	demo.SetLocalDB(kvdb)
	addr := &types.ReqAddr{Addr: "1HUiTRFvp6HvW6eacgV9EoBSgroRDiUsMs", Count: 1, Direction: 1}
	kvdb.Set(types.CalcTxAddrHashKey(addr.GetAddr(), ""), types.Encode(&types.ReplyTxInfo{Height: 1}))
	_, err := demo.GetTxsByAddr(addr)
	assert.Equal(t, types.ErrNotFound, err)

	addr.Height = -1
	_, err = demo.GetTxsByAddr(addr)
	assert.Equal(t, nil, err)

	c, err := demo.GetPrefixCount(&types.ReqKey{Key: types.CalcTxAddrHashKey(addr.GetAddr(), "")})
	assert.NoError(t, err)
	assert.True(t, c.(*types.Int64).Data == 1)

	_, err = demo.GetAddrTxsCount(&types.ReqKey{Key: types.CalcTxAddrHashKey(addr.GetAddr(), "")})
	assert.NoError(t, err)

	_, err = demo.Query("", nil)
	assert.Equal(t, types.ErrActionNotSupport, err)
}

// Below both fork heights a legacy address form is tolerated, at and above them it is
// rejected. The gate has to agree with how the blocks on the chain were executed: at
// 546820 -- below bityuan's 2270000 -- a transaction to this address was accepted, and a
// sync that rejects it cannot reproduce that block's state root.
func TestCheckAddressToleratesLegacyFormBelowFork(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	const fork = int64(2270000)
	cfg.SetFork("ForkMultiSignAddress", fork)
	cfg.SetFork("ForkBase58AddressCheck", fork)

	legacy := "DsYQcck3QFK9Wt1UWd5eoskWjk8JdYSCMoK"
	assert.Equal(t, address.ErrCheckVersion, address.CheckAddress(legacy, fork-1),
		"the raw check reports the version mismatch the gate matches on")
	assert.NoError(t, CheckAddress(cfg, legacy, fork-1))
	assert.Error(t, CheckAddress(cfg, legacy, fork))
}

// A 25-byte address whose checksum fails stays rejected below the fork too. The chain
// executed such a transaction as a failure -- block 101641 packed this transfer as
// ExecPack with "Address Checksum error" -- so tolerating ErrCheckChecksum replays it as
// a success instead. Only ErrAddressChecksum, the longer form above, is tolerated below
// ForkBase58AddressCheck.
func TestCheckAddressRejectsCorruptChecksumBelowFork(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	const fork = int64(100)
	cfg.SetFork("ForkMultiSignAddress", fork)
	cfg.SetFork("ForkBase58AddressCheck", fork)

	// block 101641's coins recipient: 25 bytes, correct version byte, wrong checksum
	corrupt := "1Di16bUjPJnvZ8Hrf4vuQDffzkv9jC5Jp"
	assert.Equal(t, address.ErrCheckChecksum, address.CheckAddress(corrupt, fork-1))
	assert.Equal(t, address.ErrCheckChecksum, CheckAddress(cfg, corrupt, fork-1))
	assert.Equal(t, address.ErrCheckChecksum, CheckAddress(cfg, corrupt, fork))
}

// The two fork heights are separate knobs, and they are only equal on chains that say so
// (bityuan sets both to 2270000; the chain33 defaults are 1298600 and 1800000). This pins
// the window between them, where the version error stops being tolerated while the checksum
// error still is: an address that fails only the version check is rejected from
// ForkMultiSignAddress up, which is the behaviour change #1401 disclosed and left in place.
// The 25-byte checksum case does not depend on ForkMultiSignAddress at all.
func TestCheckAddressSeparatesTheTwoForkHeights(t *testing.T) {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	cfg.SetFork("ForkMultiSignAddress", 100)
	cfg.SetFork("ForkBase58AddressCheck", 200)

	// Fails only the version check.
	version := "DsYQcck3QFK9Wt1UWd5eoskWjk8JdYSCMoK"
	assert.NoError(t, CheckAddress(cfg, version, 99))
	assert.Equal(t, address.ErrCheckVersion, CheckAddress(cfg, version, 100))
	assert.Equal(t, address.ErrCheckVersion, CheckAddress(cfg, version, 199))
	assert.Equal(t, address.ErrCheckVersion, CheckAddress(cfg, version, 200))

	// Fails the 25-byte checksum: rejected at every height, in both windows.
	corrupt := "1Di16bUjPJnvZ8Hrf4vuQDffzkv9jC5Jp"
	for _, height := range []int64{99, 100, 199, 200} {
		assert.Equal(t, address.ErrCheckChecksum, CheckAddress(cfg, corrupt, height),
			"25-byte checksum failure tolerated at height %d", height)
	}
}
