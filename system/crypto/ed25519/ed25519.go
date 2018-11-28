// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package ed25519 ed25519系统加密包
package ed25519

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/common/ed25519"
)

//Driver 驱动
type Driver struct{}

//GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], crypto.CRandBytes(32))
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes), nil
}

//PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != 64 {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], b[:32])
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes), nil
}

//PubKeyFromBytes 字节转为公钥
func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != 32 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([32]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeyEd25519(*pubKeyBytes), nil
}

//SignatureFromBytes 字节转为签名
func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	sigBytes := new([64]byte)
	copy(sigBytes[:], b[:])
	return SignatureEd25519(*sigBytes), nil
}

//PrivKeyEd25519 PrivKey
type PrivKeyEd25519 [64]byte

//Bytes 字节格式
func (privKey PrivKeyEd25519) Bytes() []byte {
	s := make([]byte, 64)
	copy(s, privKey[:])
	return s
}

//Sign 签名
func (privKey PrivKeyEd25519) Sign(msg []byte) crypto.Signature {
	privKeyBytes := [64]byte(privKey)
	signatureBytes := ed25519.Sign(&privKeyBytes, msg)
	return SignatureEd25519(*signatureBytes)
}

//PubKey 公钥
func (privKey PrivKeyEd25519) PubKey() crypto.PubKey {
	privKeyBytes := [64]byte(privKey)
	return PubKeyEd25519(*ed25519.MakePublicKey(&privKeyBytes))
}

//Equals 相等
func (privKey PrivKeyEd25519) Equals(other crypto.PrivKey) bool {
	if otherEd, ok := other.(PrivKeyEd25519); ok {
		return bytes.Equal(privKey[:], otherEd[:])
	}
	return false

}

//PubKeyEd25519 PubKey
type PubKeyEd25519 [32]byte

//Bytes 字节格式
func (pubKey PubKeyEd25519) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, pubKey[:])
	return s
}

//VerifyBytes 验证字节
func (pubKey PubKeyEd25519) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	// unwrap if needed
	if wrap, ok := sig.(SignatureS); ok {
		sig = wrap.Signature
	}
	// make sure we use the same algorithm to sign
	sigEd25519, ok := sig.(SignatureEd25519)
	if !ok {
		return false
	}
	pubKeyBytes := [32]byte(pubKey)
	sigBytes := [64]byte(sigEd25519)
	return ed25519.Verify(&pubKeyBytes, msg, &sigBytes)
}

//KeyString 公钥字符串格式
func (pubKey PubKeyEd25519) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

//Equals 相等
func (pubKey PubKeyEd25519) Equals(other crypto.PubKey) bool {
	if otherEd, ok := other.(PubKeyEd25519); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	}
	return false

}

//SignatureEd25519 Signature
type SignatureEd25519 [64]byte

//SignatureS 签名
type SignatureS struct {
	crypto.Signature
}

//Bytes 字节格式
func (sig SignatureEd25519) Bytes() []byte {
	s := make([]byte, 64)
	copy(s, sig[:])
	return s
}

//IsZero 是否是0
func (sig SignatureEd25519) IsZero() bool { return len(sig) == 0 }

func (sig SignatureEd25519) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)
}

//Equals 相等
func (sig SignatureEd25519) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureEd25519); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false

}

//const
const (
	Name = "ed25519"
	ID   = 2
)

func init() {
	crypto.Register(Name, &Driver{})
	crypto.RegisterType(Name, ID)
}
