// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package sm2 系统签名包
package sm2

import (
	"bytes"
	"errors"
	"fmt"

	"crypto/elliptic"
	"encoding/asn1"
	"math/big"

	"github.com/33cn/chain33/common/crypto"
	"github.com/tjfoc/gmsm/sm2"
)

//const
const (
	SM2PrivateKeyLength = 32
	SM2PublicKeyLength  = 65
)

//Driver 驱动
type Driver struct{}

//GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [SM2PrivateKeyLength]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(SM2PrivateKeyLength))
	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKeyBytes[:])
	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeySM2(privKeyBytes), nil
}

//PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != SM2PrivateKeyLength {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([SM2PrivateKeyLength]byte)
	copy(privKeyBytes[:], b[:SM2PrivateKeyLength])

	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKeyBytes[:])

	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeySM2(*privKeyBytes), nil
}

//PubKeyFromBytes 字节转为公钥
func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != SM2PublicKeyLength {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([SM2PublicKeyLength]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySM2(*pubKeyBytes), nil
}

//SignatureFromBytes 字节转为签名
func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	var certSignature crypto.CertSignature
	_, err = asn1.Unmarshal(b, &certSignature)
	if err != nil {
		return SignatureSM2(b), nil
	}

	if len(certSignature.Cert) == 0 {
		return SignatureSM2(b), nil
	}

	return SignatureSM2(certSignature.Signature), nil
}

//PrivKeySM2 私钥
type PrivKeySM2 [SM2PrivateKeyLength]byte

//Bytes 字节格式
func (privKey PrivKeySM2) Bytes() []byte {
	s := make([]byte, SM2PrivateKeyLength)
	copy(s, privKey[:])
	return s
}

//Sign 签名
func (privKey PrivKeySM2) Sign(msg []byte) crypto.Signature {
	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKey[:])
	r, s, err := sm2.Sign(priv, crypto.Sm3Hash(msg))
	if err != nil {
		return nil
	}

	//sm2不需要LowS转换
	//s = ToLowS(pub, s)
	return SignatureSM2(Serialize(r, s))
}

//PubKey 私钥生成公钥
func (privKey PrivKeySM2) PubKey() crypto.PubKey {
	_, pub := privKeyFromBytes(sm2.P256Sm2(), privKey[:])
	var pubSM2 PubKeySM2
	copy(pubSM2[:], SerializePublicKey(pub))
	return pubSM2
}

//Equals 公钥
func (privKey PrivKeySM2) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySM2); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	}

	return false
}

func (privKey PrivKeySM2) String() string {
	return fmt.Sprintf("PrivKeySM2{*****}")
}

//PubKeySM2 公钥
type PubKeySM2 [SM2PublicKeyLength]byte

//Bytes 字节格式
func (pubKey PubKeySM2) Bytes() []byte {
	s := make([]byte, SM2PublicKeyLength)
	copy(s, pubKey[:])
	return s
}

//VerifyBytes 验证字节
func (pubKey PubKeySM2) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	if wrap, ok := sig.(SignatureS); ok {
		sig = wrap.Signature
	}

	sigSM2, ok := sig.(SignatureSM2)
	if !ok {
		fmt.Printf("convert failed\n")
		return false
	}

	pub, err := parsePubKey(pubKey[:], sm2.P256Sm2())
	if err != nil {
		fmt.Printf("parse pubkey failed\n")
		return false
	}

	r, s, err := Deserialize(sigSM2)
	if err != nil {
		fmt.Printf("unmarshal sign failed")
		return false
	}

	//国密签名算法和ecdsa不一样，-s验签不通过，所以不需要LowS检查
	//fmt.Printf("verify:%x, r:%d, s:%d\n", crypto.Sm3Hash(msg), r, s)
	//lowS := IsLowS(s)
	//if !lowS {
	//	fmt.Printf("lowS check failed")
	//	return false
	//}

	return sm2.Verify(pub, crypto.Sm3Hash(msg), r, s)
}

func (pubKey PubKeySM2) String() string {
	return fmt.Sprintf("PubKeySM2{%X}", pubKey[:])
}

//KeyString Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySM2) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

//Equals 相等
func (pubKey PubKeySM2) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySM2); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	}
	return false

}

//SignatureSM2 签名
type SignatureSM2 []byte

//SignatureS 签名
type SignatureS struct {
	crypto.Signature
}

//Bytes 字节格式
func (sig SignatureSM2) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

//IsZero 是否为0
func (sig SignatureSM2) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSM2) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

//Equals 相等
func (sig SignatureSM2) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureSM2); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false
}

//const
const (
	Name = "sm2"
	ID   = 3
)

func init() {
	crypto.Register(Name, &Driver{})
	crypto.RegisterType(Name, ID)
}

func privKeyFromBytes(curve elliptic.Curve, pk []byte) (*sm2.PrivateKey, *sm2.PublicKey) {
	x, y := curve.ScalarBaseMult(pk)

	priv := &sm2.PrivateKey{
		PublicKey: sm2.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}

	return priv, &priv.PublicKey
}
