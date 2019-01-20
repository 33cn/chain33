// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package common

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/33cn/chain33/common/crypto/sha3"
	"golang.org/x/crypto/ripemd160"
)

func hexEncode(b []byte) string {
	enc := make([]byte, len(b)*2+2)
	copy(enc, "0x")
	hex.Encode(enc[2:], b)
	return string(enc)
}

//ToHex []byte -> hex
func ToHex(b []byte) string {
	hex := Bytes2Hex(b)
	// Prefer output of "0x0" instead of "0x"
	if len(hex) == 0 {
		return ""
	}
	return "0x" + hex
}

//HashHex []byte -> hex
func HashHex(d []byte) string {
	var buf [64]byte
	hex.Encode(buf[:], d)
	return string(buf[:])
}

//FromHex hex -> []byte
func FromHex(s string) ([]byte, error) {
	if len(s) > 1 {
		if s[0:2] == "0x" || s[0:2] == "0X" {
			s = s[2:]
		}
		if len(s)%2 == 1 {
			s = "0" + s
		}
		return Hex2Bytes(s)
	}
	return []byte{}, nil
}

// CopyBytes Returns an exact copy of the provided bytes
func CopyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)

	return
}

//HasHexPrefix 是否包含0x前缀
func HasHexPrefix(str string) bool {
	l := len(str)
	return l >= 2 && str[0:2] == "0x"
}

//IsHex 是否是hex字符串
func IsHex(str string) bool {
	l := len(str)
	return l >= 4 && l%2 == 0 && str[0:2] == "0x"
}

//Bytes2Hex []byte -> hex
func Bytes2Hex(d []byte) string {
	return hex.EncodeToString(d)
}

//Sha256 加密
func Sha256(b []byte) []byte {
	data := sha256.Sum256(b)
	return data[:]
}

//ShaKeccak256 加密
func ShaKeccak256(b []byte) []byte {
	data := sha3.KeccakSum256(b)
	return data[:]
}

//Hex2Bytes hex -> []byte
func Hex2Bytes(str string) ([]byte, error) {
	return hex.DecodeString(str)
}

func sha2Hash(b []byte, out []byte) {
	s := sha256.New()
	s.Write(b[:])
	tmp := s.Sum(nil)
	s.Reset()
	s.Write(tmp)
	copy(out[:], s.Sum(nil))
}

// Sha2Sum Returns hash: SHA256( SHA256( data ) )
// Where possible, using ShaHash() should be a bit faster
func Sha2Sum(b []byte) (out [32]byte) {
	sha2Hash(b, out[:])
	return
}

func rimpHash(in []byte, out []byte) {
	sha := sha256.New()
	sha.Write(in)
	rim := ripemd160.New()
	rim.Write(sha.Sum(nil)[:])
	copy(out, rim.Sum(nil))
}

// Rimp160AfterSha256 Returns hash: RIMP160( SHA256( data ) )
// Where possible, using RimpHash() should be a bit faster
func Rimp160AfterSha256(b []byte) (out [20]byte) {
	rimpHash(b, out[:])
	return
}
