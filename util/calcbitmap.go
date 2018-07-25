package util

import (
	"bytes"

	"gitlab.33.cn/chain33/chain33/types"
)

//big-end mode, that is byte [0]      [1]
// 				tx index:    01234567 89abcdef
//cur is subset of ori
func CalcByteBitMap(ori, cur [][]byte, data []*types.ReceiptData) []byte {
	var bitRst byte
	var rst []byte
	for index := 0; index < len(ori); index++ {
		if index > 0 && index%8 == 0 {
			rst = append(rst, bitRst)
			bitRst = 0
		}
		for i, curHash := range cur {
			if bytes.Equal(ori[index], curHash) {
				if data[i].Ty == types.ExecOk {
					bitRst |= 1 << (7 - (uint32(index) % 8))
				}
			}
		}
	}

	if len(ori)%8 > 0 {
		rst = append(rst, bitRst)
	}
	return rst
}

//index begin from 0
func DecodeByteBitMap(bitmap []byte, index uint32) bool {
	n := index / 8
	i := index % 8
	return (0x1 & (bitmap[n] >> (7 - i))) == 0x1
}
