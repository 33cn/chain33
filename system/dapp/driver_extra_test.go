package dapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type testDriver struct {
	*DriverBase
}

func newTestDriver() *testDriver {
	t := &testDriver{DriverBase: &DriverBase{}}
	t.SetChild(t)
	return t
}

func (t *testDriver) GetDriverName() string { return "test" }

func TestDriverBaseUpgrade(t *testing.T) {
	d := newTestDriver()
	result, err := d.Upgrade()
	assert.Nil(t, err)
	assert.Nil(t, result)
}

func TestDriverBaseCheckReceiptExecOk(t *testing.T) {
	d := newTestDriver()
	assert.False(t, d.CheckReceiptExecOk())
}

func TestDriverBaseIsFree(t *testing.T) {
	d := newTestDriver()
	assert.False(t, d.IsFree())
	d.SetIsFree(true)
	assert.True(t, d.IsFree())
}

func TestDriverBaseExecutorOrder(t *testing.T) {
	d := newTestDriver()
	assert.True(t, d.ExecutorOrder() == 0)
}
