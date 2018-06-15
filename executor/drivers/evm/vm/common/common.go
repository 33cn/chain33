package common

import (
	"math"
	"math/big"
)

// 返回从开始位置制定长度的数据
// 如果源数据不足，则剩余位置用零填充
func GetData(data []byte, start uint64, size uint64) []byte {
	length := uint64(len(data))
	if start > length {
		start = length
	}
	end := start + size
	if end > length {
		end = length
	}
	return RightPadBytes(data[start:end], int(size))
}

// 返回从开始位置制定长度的数据
// 如果源数据不足，则剩余位置用零填充
func GetDataBig(data []byte, start *big.Int, size *big.Int) []byte {
	dlen := big.NewInt(int64(len(data)))

	s := BigMin(start, dlen)
	e := BigMin(new(big.Int).Add(s, size), dlen)
	return RightPadBytes(data[s.Uint64():e.Uint64()], int(size.Uint64()))
}

// 将大整数转换为uint64，并判断是否溢出
func BigUint64(v *big.Int) (uint64, bool) {
	return v.Uint64(), v.BitLen() > 64
}

// 计算制定字节长度所对应的字长度（一个字，对应32个字节，也就是256位）
// 如果长度不足一个字长，则补足
// EVM在内存中存储时的最小单位是字长（256位），而不是字节
func ToWordSize(size uint64) uint64 {
	if size > math.MaxUint64-31 {
		return math.MaxUint64/32 + 1
	}

	return (size + 31) / 32
}

// 判断字节数组内容是否全为零
func AllZero(b []byte) bool {
	for _, byte := range b {
		if byte != 0 {
			return false
		}
	}
	return true
}
