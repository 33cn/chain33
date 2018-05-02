package common

import "encoding/hex"

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

// 字节数组转换为十六进制字符串表示
func ToHex(b []byte) string {
	hex := Bytes2Hex(b)

	if len(hex) == 0 {
		hex = "0"
	}
	return "0x" + hex
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

// 字节数组转换为十六进制字符串表示
func Bytes2Hex(d []byte) string {
	return hex.EncodeToString(d)
}

// 十六进制字符串转换为字节数组
func Hex2Bytes(str string) []byte {
	h, _ := hex.DecodeString(str)
	return h
}
