// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package merkle

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPow2(t *testing.T) {
	assert.Equal(t, 1, pow2(0))
	assert.Equal(t, 2, pow2(1))
	assert.Equal(t, 4, pow2(2))
	assert.Equal(t, 8, pow2(3))
	assert.Equal(t, 1024, pow2(10))
}

func TestDecodeHash(t *testing.T) {
	var h Hash
	err := Decode(&h, "")
	assert.Nil(t, err)

	err = Decode(&h, "0a0b")
	assert.Nil(t, err)

	longStr := ""
	for i := 0; i < MaxHashStringSize+2; i++ {
		longStr += "a"
	}
	err = Decode(&h, longStr)
	assert.Equal(t, ErrHashStrSize, err)
}

func TestSetBytes(t *testing.T) {
	var h Hash
	err := h.SetBytes(nil)
	assert.NotNil(t, err)

	err = h.SetBytes([]byte{})
	assert.NotNil(t, err)

	validBytes := make([]byte, 32)
	err = h.SetBytes(validBytes)
	assert.Nil(t, err)
}

func TestNewHash(t *testing.T) {
	h, err := NewHash(nil)
	assert.NotNil(t, err)
	assert.Nil(t, h)

	valid := make([]byte, 32)
	h, err = NewHash(valid)
	assert.Nil(t, err)
	assert.NotNil(t, h)
}

func TestNewHashFromStr(t *testing.T) {
	h, err := NewHashFromStr("")
	assert.Nil(t, err)
	assert.NotNil(t, h)

	validHex := "0a0b0c0d0e0f101112131415161718191a1b1c1d1e1f202122232425262728292a2b2c2d2e2f30"
	h, err = NewHashFromStr(validHex[:64])
	assert.Nil(t, err)
	assert.NotNil(t, h)
}
