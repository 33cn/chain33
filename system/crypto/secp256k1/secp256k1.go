// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package secp256k1 secp256k1系统加密包
package secp256k1

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/33cn/chain33/common/crypto"
	secp256k1 "github.com/btcsuite/btcd/btcec"
)

//Driver 驱动
type Driver struct{}

//GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(32))
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return PrivKeySecp256k1(privKeyBytes), nil
}

//PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != 32 {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([32]byte)
	copy(privKeyBytes[:], b[:32])
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return PrivKeySecp256k1(*privKeyBytes), nil
}

//PubKeyFromBytes 字节转为公钥
func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != 33 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([33]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySecp256k1(*pubKeyBytes), nil
}

//SignatureFromBytes 字节转为签名
func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	return SignatureSecp256k1(b), nil
}

//PrivKeySecp256k1 PrivKey
type PrivKeySecp256k1 [32]byte

//Bytes 字节格式
func (privKey PrivKeySecp256k1) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, privKey[:])
	return s
}

//Sign 签名
func (privKey PrivKeySecp256k1) Sign(msg []byte) crypto.Signature {
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey[:])
	sig, err := priv.Sign(crypto.Sha256(msg))
	if err != nil {
		panic("Error signing secp256k1" + err.Error())
	}
	return SignatureSecp256k1(sig.Serialize())
}

//PubKey 私钥生成公钥
func (privKey PrivKeySecp256k1) PubKey() crypto.PubKey {
	_, pub := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey[:])
	var pubSecp256k1 PubKeySecp256k1
	copy(pubSecp256k1[:], pub.SerializeCompressed())
	return pubSecp256k1
}

//Equals 私钥是否相等
func (privKey PrivKeySecp256k1) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySecp256k1); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	}
	return false

}

func (privKey PrivKeySecp256k1) String() string {
	return fmt.Sprintf("PrivKeySecp256k1{*****}")
}

// PubKey

//PubKeySecp256k1 Compressed pubkey (just the x-cord),
// prefixed with 0x02 or 0x03, depending on the y-cord.
type PubKeySecp256k1 [33]byte

//Bytes 字节格式
func (pubKey PubKeySecp256k1) Bytes() []byte {
	s := make([]byte, 33)
	copy(s, pubKey[:])
	return s
}

//VerifyBytes 验证字节
func (pubKey PubKeySecp256k1) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	// unwrap if needed
	if wrap, ok := sig.(SignatureS); ok {
		sig = wrap.Signature
	}
	// and assert same algorithm to sign and verify
	sigSecp256k1, ok := sig.(SignatureSecp256k1)
	if !ok {
		return false
	}

	pub, err := secp256k1.ParsePubKey(pubKey[:], secp256k1.S256())
	if err != nil {
		return false
	}
	sig2, err := secp256k1.ParseDERSignature(sigSecp256k1[:], secp256k1.S256())
	if err != nil {
		return false
	}
	return sig2.Verify(crypto.Sha256(msg), pub)
}

func (pubKey PubKeySecp256k1) String() string {
	return fmt.Sprintf("PubKeySecp256k1{%X}", pubKey[:])
}

//KeyString Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySecp256k1) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

//Equals 公钥相等
func (pubKey PubKeySecp256k1) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySecp256k1); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	}
	return false

}

//SignatureSecp256k1 Signature
type SignatureSecp256k1 []byte

//SignatureS 签名
type SignatureS struct {
	crypto.Signature
}

//Bytes 字节格式
func (sig SignatureSecp256k1) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

//IsZero 是否是0
func (sig SignatureSecp256k1) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSecp256k1) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

//Equals 相等
func (sig SignatureSecp256k1) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureSecp256k1); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false

}

//const
const (
	Name = "secp256k1"
	ID   = 1
)

func init() {
	crypto.Register(Name, &Driver{})
	crypto.RegisterType(Name, ID)
}
