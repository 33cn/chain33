package tests

import (
	"testing"

	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
)

type Tester struct {
	t *testing.T
}

func NewTester(t *testing.T) *Tester {
	return &Tester{t: t}
}

func (t *Tester) assertNil(val interface{}) {
	if val != nil {
		t.t.Errorf("value {%s} is not nil", val)
		t.t.Fail()
	}
}

func (t *Tester) assertNilB(val []byte) {
	if val != nil {
		t.t.Errorf("value {%s} is not nil", common.Bytes2Hex(val))
		t.t.Fail()
	}
}

func (t *Tester) assertNotNil(val interface{}) {
	if val == nil {
		t.t.Errorf("value {%s} is nil", val)
		t.t.Fail()
	}
}

func (t *Tester) assertEquals(val1, val2 struct{}) {
	if val1 != val2 {
		t.t.Errorf("value {%s} is not equals to {%s}", val1, val2)
		t.t.Fail()
	}
}

func (t *Tester) assertEqualsS(val1, val2 string) {
	if val1 != val2 {
		t.t.Errorf("value %v is not equals to %v", val1, val2)
		t.t.Fail()
	}
}

func (t *Tester) assertEqualsV(val1, val2 int) {
	if val1 != val2 {
		t.t.Errorf("value %v is not equals to %v", val1, val2)
		t.t.Fail()
	}
}
func (t *Tester) assertEqualsE(val1, val2 error) {
	if val1 != val2 {
		t.t.Errorf("value %v is not equals to %v", val1, val2)
		t.t.Fail()
	}
}

func (t *Tester) assertEqualsB(val1, val2 []byte) {
	if string(val1) != string(val2) {
		t.t.Errorf("value %v is not equals to %v", common.Bytes2Hex(val1), common.Bytes2Hex(val2))
		t.t.Fail()
	}
}

func (t *Tester) assertBigger(val1, val2 int) {
	if val1 < val2 {
		t.t.Errorf("value %v is less than %v", val1, val2)
		t.t.Fail()
	}
}

func (t *Tester) assertNotEquals(val1, val2 struct{}) {
	if val1 == val2 {
		t.t.Errorf("value %v is equals to %v", val1, val2)
		t.t.Fail()
	}
}

func (t *Tester) assertNotEqualsI(val1, val2 interface{}) {
	if val1 == val2 {
		t.t.Errorf("value %v is equals to %v", val1, val2)
		t.t.Fail()
	}
}
func (t *Tester) assertNotEqualsV(val1, val2 int) {
	if val1 == val2 {
		t.t.Errorf("value %v is equals to %v", val1, val2)
		t.t.Fail()
	}
}
