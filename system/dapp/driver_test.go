package dapp

import (
	"testing"
	"time"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/rpc/grpcclient"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

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

func init() {
	Register("none", newnoneApp, 0)
	Register("demo", newdemoApp, 1)
}

func TestReigister(t *testing.T) {
	_, err := LoadDriver("demo", 0)
	assert.Equal(t, err, types.ErrUnknowDriver)
	_, err = LoadDriver("demo", 1)
	assert.Equal(t, err, nil)

	tx := &types.Transaction{Execer: []byte("demo")}
	driver := LoadDriverAllow(tx, 0, 0)
	assert.Equal(t, "none", driver.GetDriverName())
	driver = LoadDriverAllow(tx, 0, 1)
	assert.Equal(t, "demo", driver.GetDriverName())

	types.SetTitleOnlyForTest("user.p.hello.")
	tx = &types.Transaction{Execer: []byte("demo")}
	driver = LoadDriverAllow(tx, 0, 0)
	assert.Equal(t, "none", driver.GetDriverName())
	driver = LoadDriverAllow(tx, 0, 1)
	assert.Equal(t, "demo", driver.GetDriverName())

	tx.Execer = []byte("user.p.hello.demo")
	driver = LoadDriverAllow(tx, 0, 1)
	assert.Equal(t, "demo", driver.GetDriverName())

	tx.Execer = []byte("user.p.hello2.demo")
	driver = LoadDriverAllow(tx, 0, 1)
	assert.Equal(t, "none", driver.GetDriverName())
}

func TestDriverAPI(t *testing.T) {
	tx := &types.Transaction{Execer: []byte("demo")}
	demo := LoadDriverAllow(tx, 0, 1).(*demoApp)
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	demo.SetEnv(1, time.Now().Unix(), 1)
	demo.SetBlockInfo([]byte("parentHash"), []byte("mainHash"), 1)
	demo.SetLocalDB(kvdb)
	demo.SetStateDB(kvdb)
	demo.SetAPI(&mocks.QueueProtocolAPI{})
	gcli, err := grpcclient.NewMainChainClient("")
	assert.Nil(t, err)
	demo.SetExecutorAPI(&mocks.QueueProtocolAPI{}, gcli)
	assert.NotNil(t, demo.GetAPI())
	assert.NotNil(t, demo.GetExecutorAPI())
	types.SetTitleOnlyForTest("chain33")
	assert.Equal(t, "parentHash", string(demo.GetParentHash()))
	assert.Equal(t, "parentHash", string(demo.GetLastHash()))
	types.SetTitleOnlyForTest("user.p.wzw.")
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
	tx := &types.Transaction{Execer: []byte("demo")}
	demo := LoadDriverAllow(tx, 0, 1).(*demoApp)
	assert.Equal(t, true, demo.AllowIsSame([]byte("demo")))
	types.SetTitleOnlyForTest("user.p.wzw.")
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
	dir, ldb, kvdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, ldb)
	demo := newdemoApp().(*demoApp)
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

	err = CheckAddress("1HUiTRFvp6HvW6eacgV9EoBSgroRDiUsMs", 0)
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

	_, err = demo.DelRollbackKV(tx, []byte(execer))
	assert.Equal(t, err, types.ErrNotFound)

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
	kvdb.Set(types.CalcTxAddrHashKey(addr.GetAddr(), ""), types.Encode(&types.ReplyTxInfo{}))
	_, err := demo.GetTxsByAddr(addr)
	assert.Equal(t, types.ErrNotFound, err)

	addr.Height = -1
	_, err = demo.GetTxsByAddr(addr)
	assert.NoError(t, err)

	c, err := demo.GetPrefixCount(&types.ReqKey{Key: types.CalcTxAddrHashKey(addr.GetAddr(), "")})
	assert.NoError(t, err)
	assert.True(t, c.(*types.Int64).Data == 1)

	_, err = demo.GetAddrTxsCount(&types.ReqKey{Key: types.CalcTxAddrHashKey(addr.GetAddr(), "")})
	assert.NoError(t, err)

	_, err = demo.Query("", nil)
	assert.Equal(t, types.ErrActionNotSupport, err)
}
