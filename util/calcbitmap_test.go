package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestCalcByteBitMap(t *testing.T) {
	ori := [][]byte{} //{0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17}
	for i := 0; i < 18; i++ {
		ori = append(ori, []byte{byte(i)})
	}
	cur := [][]byte{}
	arry := []byte{3, 7, 8, 11, 15, 17}
	for _, v := range arry {
		cur = append(cur, []byte{byte(v)})
	}

	d0 := &types.ReceiptData{Ty: types.ExecOk}
	d1 := &types.ReceiptData{Ty: types.ExecPack}
	d2 := &types.ReceiptData{Ty: types.ExecOk}
	d3 := &types.ReceiptData{Ty: types.ExecOk}
	d4 := &types.ReceiptData{Ty: types.ExecOk}
	d5 := &types.ReceiptData{Ty: types.ExecOk}
	data := []*types.ReceiptData{d0, d1, d2, d3, d4, d5}

	//     {0,1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17}
	//rst:  0,0,0,1,0,0,0,0,1,0,0, 1, 0, 0, 0, 1, 0, 1
	//      16              145                   64
	rst := CalcByteBitMap(ori, cur, data)
	t.Log(rst)
	check := []byte{16, 145, 64}
	assert.Equal(t, check, rst)
}

func TestDecodeByteBitMap(t *testing.T) {
	var i uint32
	rst := []byte{16, 145, 64}
	i = 2
	ret := DecodeByteBitMap(rst, i)
	assert.False(t, ret)

	i = 3
	ret = DecodeByteBitMap(rst, i)
	assert.True(t, ret)

	i = 8
	ret = DecodeByteBitMap(rst, i)
	assert.True(t, ret)
}
