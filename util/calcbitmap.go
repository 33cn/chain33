// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"bytes"
	"math/big"

	"github.com/33cn/chain33/types"
)

//CalcBitMap big-end mode,    that is bytes [0]      [1]
// 				   tx index:     fedcba98 76543210
//receipts are align with subs txs,
// if the tx ty is OK in subs, find the tx in base and set the index to 1, this function return base's bitmap
//if all tx failed, the setBit will normalize result and just return nil slice
func CalcBitMap(bases, subs [][]byte, subData []*types.ReceiptData) []byte {
	rst := big.NewInt(0)

	for index, base := range bases {
		for i, sub := range subs {
			if bytes.Equal(base, sub) {
				if subData[i].Ty == types.ExecOk {
					rst.SetBit(rst, index, 1)
				}
			}
		}
	}

	return rst.Bytes()
}

//CalcSubBitMap : cur is subset of ori, data are align with ori, this function return cur's bitmap
//if all tx failed, the setBit will normalize result and just return nil slice
func CalcSubBitMap(ori, sub [][]byte, data []*types.ReceiptData) []byte {
	rst := big.NewInt(0)

	for i, subHash := range sub {
		for index, ori := range ori {
			if bytes.Equal(ori, subHash) {
				if data[index].Ty == types.ExecOk {
					rst.SetBit(rst, i, 1)
				}
			}
		}
	}

	return rst.Bytes()
}

//BitMapBit :index begin from 0, find the index bit, 1 or 0
func BitMapBit(bitmap []byte, index uint32) bool {
	rst := big.NewInt(0).SetBytes(bitmap)
	return rst.Bit(int(index)) == uint(0x1)
}
