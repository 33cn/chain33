package dapp

import (
	"testing"
	"time"

	"github.com/33cn/chain33/client/mocks"
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
