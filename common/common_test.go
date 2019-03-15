// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandString(t *testing.T) {
	str := GetRandString(10)
	t.Log(str)
	assert.Len(t, str, 10)
}

func TestInt64Add(t *testing.T) {
	i := math.MaxInt64
	assert.Equal(t, i+1, -9223372036854775808)
	assert.Equal(t, i+2, -9223372036854775807)
}

func TestPointer(t *testing.T) {
	id := StorePointer(0)
	assert.Equal(t, int64(1), id)
	data, err := GetPointer(id)
	assert.Nil(t, err)
	assert.Equal(t, 0, data)

	RemovePointer(id)
	_, err = GetPointer(id)
	assert.Equal(t, ErrPointerNotFound, err)
}

func TestRandStringLen(t *testing.T) {
	str := GetRandBytes(10, 20)
	t.Log(string(str))
	if len(str) < 10 || len(str) > 20 {
		t.Error("rand str len")
	}
}

func TestGetRandPrintString(t *testing.T) {
	str := GetRandPrintString(10, 20)
	t.Log(str)
	if len(str) < 10 || len(str) > 20 {
		t.Error("rand str len")
	}
}

func TestMinMax(t *testing.T) {
	assert.Equal(t, MinInt32(1, 2), int32(1))
	assert.Equal(t, MaxInt32(1, 2), int32(2))
}

func TestHex(t *testing.T) {
	var data [Sha256Len]byte
	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000", HashHex(data[:]))
	assert.Equal(t, 64, len(HashHex(data[:])))
	assert.Equal(t, "0x0000000000000000000000000000000000000000000000000000000000000000", ToHex(data[:]))
	bdata, err := FromHex("0x0000000000000000000000000000000000000000000000000000000000000000")
	assert.Nil(t, err)
	assert.Equal(t, bdata, data[:])

	bdata, err = FromHex("0X0000000000000000000000000000000000000000000000000000000000000000")
	assert.Nil(t, err)
	assert.Equal(t, bdata, data[:])

	bdata, err = FromHex("0000000000000000000000000000000000000000000000000000000000000000")
	assert.Nil(t, err)
	assert.Equal(t, bdata, data[:])

	data2 := CopyBytes(data[:])
	data[0] = 1
	assert.Equal(t, data2[0], uint8(0))

	assert.Equal(t, false, IsHex("0x"))
	assert.Equal(t, false, IsHex("0x0"))
	assert.Equal(t, true, IsHex("0x00"))
}

func TestHash(t *testing.T) {
	var data [Sha256Len]byte
	assert.Equal(t, "0x66687aadf862bd776c8fc18b8e9f8e20089714856ee233b3902a591d0d5f2925", ToHex(Sha256(data[:])))
	assert.Equal(t, "0x290decd9548b62a8d60345a988386fc84ba6bc95484008f6362f93160ef3e563", ToHex(Sha3(data[:])))
	assert.Equal(t, "0x2b32db6c2c0a6235fb1397e8225ea85e0f0e6e8c7b126d0016ccbde0e667151e", ToHex(Sha2Sum(data[:])))
	assert.Equal(t, "0xb8bcb07f6344b42ab04250c86a6e8b75d3fdbbc6", ToHex(Rimp160(data[:])))
}
