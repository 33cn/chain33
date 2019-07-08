// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"math/big"

	"github.com/33cn/chain33/types"
)

//CalcBitMap subs are align with subData,get the bases' tx's bitmap from subs result
// if the tx ty is OK in subs, find the tx in base and set the index to 1, this function return base's bitmap
//if all tx failed, the setBit will normalize result and just return nil slice
func CalcBitMap(bases, subs [][]byte, subData []*types.ReceiptData) []byte {
	rst := big.NewInt(0)

	subMap := make(map[string]bool)
	for i, sub := range subs {
		if subData[i].Ty == types.ExecOk {
			subMap[string(sub)] = true
		}
	}

	for i, base := range bases {
		if _, exist := subMap[string(base)]; exist {
			rst.SetBit(rst, i, 1)
		}
	}

	return rst.Bytes()
}

//CalcSingleBitMap calc bitmap to bases by data
func CalcSingleBitMap(bases [][]byte, data []*types.ReceiptData) []byte {
	rst := big.NewInt(0)

	for i := range bases {
		if data[i].Ty == types.ExecOk {
			rst.SetBit(rst, i, 1)
		}
	}

	return rst.Bytes()
}

//CalcBitMapByBitMap bitmap align with subs
func CalcBitMapByBitMap(bases, subs [][]byte, bitmap []byte) []byte {
	rst := big.NewInt(0)
	bit := big.NewInt(0).SetBytes(bitmap)

	subMap := make(map[string]bool)
	for i, sub := range subs {
		if bit.Bit(i) == uint(0x1) {
			subMap[string(sub)] = true
		}
	}

	for i, base := range bases {
		if _, exist := subMap[string(base)]; exist {
			rst.SetBit(rst, i, 1)
		}
	}

	return rst.Bytes()
}

//BitMapBit :index begin from 0, find the index bit, 1 or 0
func BitMapBit(bitmap []byte, index uint32) bool {
	rst := big.NewInt(0).SetBytes(bitmap)
	return rst.Bit(int(index)) == uint(0x1)
}
