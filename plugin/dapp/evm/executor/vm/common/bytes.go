package common

import (
	"encoding/hex"
	"sort"
)

// 右填充字节数组
func RightPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded, slice)

	return padded
}

// 左填充字节数组
func LeftPadBytes(slice []byte, l int) []byte {
	if l <= len(slice) {
		return slice
	}

	padded := make([]byte, l)
	copy(padded[l-len(slice):], slice)

	return padded
}

// 十六进制的字符串转换为字节数组
func FromHex(s string) []byte {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
	}
	if len(s)%2 == 1 {
		s = "0" + s
	}
	return Hex2Bytes(s)
}

// 十六进制字符串转换为字节数组
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}

// 将字节数组转换为16进制的字符串表示
func Bytes2Hex(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

// 将字节数组转换为16进制的字符串表示
// 并且将前面多余的0去除
func Bytes2HexTrim(b []byte) string {
	// 获取字节数组中第一个非零字节位置
	idx := sort.Search(len(b), func(i int) bool {
		return b[i] != 0
	})

	// 如果全0，需要特殊处理，避免值返回0x
	if idx == len(b) {
		return "0x00"
	}
	data := b[idx:]
	enc := make([]byte, len(data)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], data)
	return string(enc)
}
