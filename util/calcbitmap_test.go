// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestCalcByteBitMap(t *testing.T) {
	ori := [][]byte{} //{0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17}
	for i := 0; i < 18; i++ {
		ori = append(ori, common.Sha256([]byte(string(i))))
	}
	cur := [][]byte{}
	arry := []byte{3, 7, 8, 11, 15, 17}
	for _, v := range arry {
		cur = append(cur, common.Sha256([]byte(string(v))))
	}

	d0 := &types.ReceiptData{Ty: types.ExecOk}
	d1 := &types.ReceiptData{Ty: types.ExecPack}
	d2 := &types.ReceiptData{Ty: types.ExecOk}
	d3 := &types.ReceiptData{Ty: types.ExecOk}
	d4 := &types.ReceiptData{Ty: types.ExecOk}
	d5 := &types.ReceiptData{Ty: types.ExecOk}
	data := []*types.ReceiptData{d0, d1, d2, d3, d4, d5}

	//     {17,16,15,14,13,12,11,10,9,8,7,6,5,4,3,2,1,0}
	//rst:  1, 0, 1, 0, 0, 0, 1, 0, 0,1,0,0,0,0,1,0,0,0
	//      2     0x89                  8
	rst := CalcBitMap(ori, cur, data)
	//t.Log(rst)
	check := []byte{0x2, 0x89, 0x8}
	assert.Equal(t, check, rst)
}

func TestCalcSubBitMap(t *testing.T) {
	ori := [][]byte{} //{0,1,2,3,4,5,6,7,8,9}
	for i := 0; i < 10; i++ {
		ori = append(ori, common.Sha256([]byte(string(i))))
	}
	sub := [][]byte{}
	arry := []byte{0, 2, 4, 6, 7, 9}
	for _, v := range arry {
		sub = append(sub, common.Sha256([]byte(string(v))))
	}

	d0 := &types.ReceiptData{Ty: types.ExecOk}
	d1 := &types.ReceiptData{Ty: types.ExecPack}
	d2 := &types.ReceiptData{Ty: types.ExecOk}
	d3 := &types.ReceiptData{Ty: types.ExecPack}
	d4 := &types.ReceiptData{Ty: types.ExecOk}
	d5 := &types.ReceiptData{Ty: types.ExecOk}
	d6 := &types.ReceiptData{Ty: types.ExecPack}
	d7 := &types.ReceiptData{Ty: types.ExecOk}
	d8 := &types.ReceiptData{Ty: types.ExecPack}
	d9 := &types.ReceiptData{Ty: types.ExecPack}
	data := []*types.ReceiptData{d0, d1, d2, d3, d4, d5, d6, d7, d8, d9}

	rst := CalcBitMap(sub, ori, data)
	//t.Log(rst)
	check := []byte{0x17}
	assert.Equal(t, check, rst)
}

func TestDecodeByteBitMap(t *testing.T) {
	var i uint32
	rst := []byte{0x2, 0x89, 0x8}
	i = 2
	ret := BitMapBit(rst, i)
	assert.False(t, ret)

	i = 3
	ret = BitMapBit(rst, i)
	assert.True(t, ret)

	i = 8
	ret = BitMapBit(rst, i)
	assert.True(t, ret)

	//test for beyond array
	i = 100
	ret = BitMapBit(rst, i)
	assert.False(t, ret)
}

func TestCalcSingleBitMap(t *testing.T) {
	ori := [][]byte{} //{0,1,2,3,4,5,6,7,8,9}
	for i := 0; i < 10; i++ {
		ori = append(ori, common.Sha256([]byte(string(i))))
	}

	d0 := &types.ReceiptData{Ty: types.ExecOk}
	d1 := &types.ReceiptData{Ty: types.ExecPack}
	d2 := &types.ReceiptData{Ty: types.ExecOk}
	d3 := &types.ReceiptData{Ty: types.ExecPack}
	d4 := &types.ReceiptData{Ty: types.ExecOk}
	d5 := &types.ReceiptData{Ty: types.ExecOk}
	d6 := &types.ReceiptData{Ty: types.ExecPack}
	d7 := &types.ReceiptData{Ty: types.ExecOk}
	d8 := &types.ReceiptData{Ty: types.ExecPack}
	d9 := &types.ReceiptData{Ty: types.ExecPack}
	data := []*types.ReceiptData{d0, d1, d2, d3, d4, d5, d6, d7, d8, d9}

	rst := CalcSingleBitMap(ori, data)
	//t.Log(rst)
	check := []byte{0xb5}
	assert.Equal(t, check, rst)

}

func TestCalcBitMapByBitMap(t *testing.T) {
	ori := [][]byte{} //{0,1,2,3,4,5,6,7,8,9}
	for i := 0; i < 10; i++ {
		ori = append(ori, common.Sha256([]byte(string(i))))
	}
	sub := [][]byte{}
	arry := []byte{0, 2, 4, 6, 7, 9}
	for _, v := range arry {
		sub = append(sub, common.Sha256([]byte(string(v))))
	}

	bitmap := []byte{0x97}
	rst := CalcBitMapByBitMap(sub, ori, bitmap)
	//t.Log(rst)
	check := []byte{0x17}
	assert.Equal(t, check, rst)
}

func TestSetAddrsBitMap(t *testing.T) {
	group := []string{"aa", "bb", "cc", "dd", "ee", "ff", "gg", "h", "i", "j"}
	addrs := []string{"aa", "ee"}

	rst, mis := SetAddrsBitMap(group, addrs)
	assert.Equal(t, []byte{0x11}, rst)
	assert.Equal(t, 0, len(mis))

	addrs = []string{"f", "ee"}
	rst, mis = SetAddrsBitMap(group, addrs)
	assert.Equal(t, []byte{0x10}, rst)
	assert.Equal(t, 1, len(mis))

	addrs = []string{"i", "j", "ee"}
	rst, mis = SetAddrsBitMap(group, addrs)
	assert.Equal(t, []byte{0x3, 0x10}, rst)
	assert.Equal(t, 0, len(mis))
}

func TestGetAddrsByBitMap(t *testing.T) {
	group := []string{"aa", "bb", "cc", "dd", "ee"}
	bitmap := []byte{0x10}

	addrs := GetAddrsByBitMap(group, bitmap)
	expect := []string{"ee"}
	assert.Equal(t, expect, addrs)

	bitmap = []byte{0x11}
	addrs = GetAddrsByBitMap(group, bitmap)
	expect = []string{"aa", "ee"}
	assert.Equal(t, expect, addrs)

	bitmap = []byte{0x1f}
	addrs = GetAddrsByBitMap(group, bitmap)
	assert.Equal(t, group, addrs)
}
