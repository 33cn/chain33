package dapp

import (
	"testing"

	"github.com/33cn/chain33/types"
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

func TestReigister(t *testing.T) {
	Register("none", newnoneApp, 0)
	Register("demo", newdemoApp, 1)
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

func TestExecAddress(t *testing.T) {
	assert.Equal(t, "16htvcBNSEA7fZhAdLJphDwQRQJaHpyHTp", ExecAddress("ticket"))
}
