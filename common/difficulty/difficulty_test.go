// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package difficulty

import (
	"math/big"
	"testing"

	"bytes"

	"github.com/33cn/chain33/common"
)

type ByteSlice []byte

func (p ByteSlice) Len() int           { return len(p) }
func (p ByteSlice) Less(i, j int) bool { return p[i] < p[j] }
func (p ByteSlice) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func TestHashToBig(t *testing.T) {
	hashstr := "0xb80b27b4a795b26b3af11340683891c6714b671054bd091d0674e07daec33772"
	hash, _ := common.FromHex(hashstr)
	bigint := HashToBig(hash)
	if len(bigint.Bytes()) != 32 {
		t.Error("Wrong hash len after HashToBig")
	}
	if !bytes.Equal(hash, bigint.Bytes()) {
		t.Error("Failed to HashToBig")
	}
}

func TestCompactToBigSmallPositive(t *testing.T) {
	var compact uint32 = 0x030000ff
	bigint := CompactToBig(compact)
	if 1 != bigint.Sign() {
		t.Error("Bad sign for compact", compact)
	}

	value := bigint.Bytes()
	if 0xff != value[0] {
		t.Errorf("Bad value for compact %x", compact)
	}

	if 1 != len(value) {
		t.Error("Bad value len for compact ", len(value))
	}
}

func TestCompactToBigBigPositive(t *testing.T) {
	var compact uint32 = 0x050000ff
	bigint := CompactToBig(compact)
	if 1 != bigint.Sign() {
		t.Error("Bad sign for compact", compact)
	}

	expectRes := []byte{0xff, 0, 0}

	value := bigint.Bytes()
	if !bytes.Equal(expectRes, value) {
		t.Errorf("Bad value for compact %x", value)
	}

	if 3 != len(value) {
		t.Error("Bad value len for compact ", len(value))
	}
}

func TestCompactToBigSmallNegative(t *testing.T) {
	var compact uint32 = 0x038000ff
	bigint := CompactToBig(compact)
	if -1 != bigint.Sign() {
		t.Error("Bad sign for compact", compact)
	}

	value := bigint.Bytes()
	if 0xff != value[0] {
		t.Errorf("Bad value for compact %x", compact)
	}

	if 1 != len(value) {
		t.Error("Bad value len for compact ", len(value))
	}
}

func TestCompactToBigBigNegative(t *testing.T) {
	var compact uint32 = 0x058000ff
	bigint := CompactToBig(compact)
	if -1 != bigint.Sign() {
		t.Error("Bad sign for compact", compact)
	}

	expectRes := []byte{0xff, 0, 0}
	value := bigint.Bytes()
	if !bytes.Equal(expectRes, value) {
		t.Errorf("Bad value for compact %x", value)
	}

	if 3 != len(value) {
		t.Error("Bad value len for compact ", len(value))
	}

	another := big.NewInt(int64(0xff0000))
	another.Neg(another)
	if 0 != another.Cmp(bigint) {
		t.Error("Different big int ")
	}
}

/////////////////////////////////////////////////////////////////
func TestBigToCompactBigNegative(t *testing.T) {
	bn := big.NewInt(int64(0xff0000))
	bn.Neg(bn)
	compact := BigToCompact(bn)
	if compact != 0x0480ff00 { //其实和0x058000ff表示的含义一致，只是移位方式不一样而已
		t.Errorf("Failed to do BigToCompact for big int:0x%x", bn.Bytes())
	}
}

func TestBigToCompactBigPositive(t *testing.T) {
	bn := big.NewInt(int64(0xff0000))
	compact := BigToCompact(bn)
	if compact != 0x0400ff00 {
		t.Errorf("Failed to do BigToCompact for big int:0x%x", bn.Bytes())
	}
}

func TestBigToCompactSmallBigNegative(t *testing.T) {
	bn := big.NewInt(int64(0xff))
	bn.Neg(bn)
	compact := BigToCompact(bn)
	if compact != 0x280ff00 {
		t.Errorf("Failed to do BigToCompact for big int:0x%x,compact:%x", bn.Bytes(), compact)
	}
}

func TestBigToCompactSmallBigPositive(t *testing.T) {
	bn := big.NewInt(int64(0xff))
	compact := BigToCompact(bn)
	if compact != 0x200ff00 {
		t.Errorf("Failed to do BigToCompact for big int:0x%x", bn.Bytes())
	}
}

func TestCalcWorkNegative(t *testing.T) {
	bigint := CalcWork(uint32(0x280ff00))
	zero := big.NewInt(0)

	if zero.Cmp(bigint) != 0 {
		t.Errorf("Failed to do CalcWork for uint32:0x%x", 0x280ff00)
	}
}

func TestCalcWork(t *testing.T) {
	bigint := CalcWork(uint32(0x0400ff00))
	byteInfo := bigint.Bytes()
	len := len(byteInfo)
	expectres := []byte{0x01, 0x01, 0x00, 0xff, 0xfe, 0xfd, 0xfd, 0xff, 0x01, 0x03,
		0x04, 0x02, 0xff, 0xfb, 0xf8, 0xf8, 0xfd, 0x04, 0x0b, 0x0e, 0x09, 0xfe, 0xf0, 0xe6, 0xe7, 0xf7, 0x10, 0x28, 0x31, 0x20}

	if len != 30 || !bytes.Equal(expectres, byteInfo) {
		t.Errorf("Failed to do CalcWork for uint32:0x%x", 0x0400ff00)
	}
}
