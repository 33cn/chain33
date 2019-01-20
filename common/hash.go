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

//Sha256Len sha256 bytes len
const Sha256Len = 32

//ToHex []byte -> hex
func ToHex(b []byte) string {
	hex := hex.EncodeToString(b)
	// Prefer output of "0x0" instead of "0x"
	if len(hex) == 0 {
		return ""
	}
	return "0x" + hex
}

//HashHex []byte -> hex
func HashHex(d []byte) string {
	var buf [Sha256Len * 2]byte
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
		return hex.DecodeString(s)
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

//IsHex 是否是hex字符串
func IsHex(str string) bool {
	l := len(str)
	return l >= 4 && l%2 == 0 && str[0:2] == "0x"
}

//Sha256 加密
func Sha256(b []byte) []byte {
	data := sha256.Sum256(b)
	return data[:]
}

//Sha3 加密
func Sha3(b []byte) []byte {
	data := sha3.KeccakSum256(b)
	return data[:]
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
func Sha2Sum(b []byte) []byte {
	out := make([]byte, 32)
	sha2Hash(b, out[:])
	return out[:]
}

func rimpHash(in []byte, out []byte) {
	sha := sha256.New()
	sha.Write(in)
	rim := ripemd160.New()
	rim.Write(sha.Sum(nil)[:])
	copy(out, rim.Sum(nil))
}

// Rimp160 Returns hash: RIMP160( SHA256( data ) )
// Where possible, using RimpHash() should be a bit faster
func Rimp160(b []byte) []byte {
	out := make([]byte, 20)
	rimpHash(b, out[:])
	return out[:]
}
