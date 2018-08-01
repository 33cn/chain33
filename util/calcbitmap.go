package util

import (
	"bytes"

	"gitlab.33.cn/chain33/chain33/types"
)

const bitPow = 3
const bitLen = 7

//big-end mode, that is byte [0]      [1]
// 				tx index:    01234567 89abcdef
//cur is subset of ori
func CalcByteBitMap(ori, cur [][]byte, data []*types.ReceiptData) []byte {
	var bitRst byte
	var rst []byte
	for index := 0; index < len(ori); index++ {
		if index > 0 && index&bitLen == 0 {
			rst = append(rst, bitRst)
			bitRst = 0
		}
		for i, curHash := range cur {
			if bytes.Equal(ori[index], curHash) {
				if data[i].Ty == types.ExecOk {
					bitRst |= 1 << (bitLen - (uint32(index) & bitLen))
				}
			}
		}
	}

	if len(ori)&bitLen > 0 {
		rst = append(rst, bitRst)
	}
	return rst
}

//index begin from 0
func DecodeByteBitMap(bitmap []byte, index uint32) bool {
	n := index >> bitPow
	i := index & bitLen
	return (0x1 & (bitmap[n] >> (bitLen - i))) == 0x1
}

func ValidBitMap(bitmap []byte, bits int) bool {
	return len(bitmap)*8 >= bits
}
