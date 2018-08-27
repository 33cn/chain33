package util

import (
	"bytes"
	"math/big"

	"gitlab.33.cn/chain33/chain33/types"
)

//big-end mode,    that is bytes [0]      [1]
// 				   tx index:     fedcba98 76543210
//cur is subset of ori, if the tx ty is OK in cur, find the tx in ori and set the index to 1
//if all tx failed, the setBit will normalize result and just return nil slice
func CalcBitMap(ori, cur [][]byte, data []*types.ReceiptData) []byte {
	rst := big.NewInt(0)

	for i, curHash := range cur {
		for index, ori := range ori {
			if bytes.Equal(ori, curHash) {
				if data[i].Ty == types.ExecOk {
					rst.SetBit(rst, index, 1)
				}
			}
		}
	}

	return rst.Bytes()
}

//index begin from 0, find the index bit, 1 or 0
func BitMapBit(bitmap []byte, index uint32) bool {
	rst := big.NewInt(0).SetBytes(bitmap)
	return rst.Bit(int(index)) == uint(0x1)
}
