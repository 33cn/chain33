// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package secp256k1sha3

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/33cn/chain33/common/crypto"
	secp256k1 "github.com/btcsuite/btcd/btcec"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const privKeyBytesLen = 32
const pubkeyBytesLen = 64 + 1

//PrivKeySecp256k1Sha3 PrivKey
type PrivKeySecp256k1Sha3 [32]byte

//Driver 驱动
type Driver struct{}

//SignatureFromBytes  对字节数组签名
func (d Driver) SignatureFromBytes(b []byte) (crypto.Signature, error) {
	return SignatureSecp256k1Sha3(b), nil
}

//PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (crypto.PrivKey, error) {
	if len(b) != privKeyBytesLen {
		return nil, errors.New("invalid priv key byte")
	}

	privKeyBytes := new([privKeyBytesLen]byte)
	copy(privKeyBytes[:], b[:privKeyBytesLen])
	return PrivKeySecp256k1Sha3(*privKeyBytes), nil
}

//PubKeyFromBytes must 65 bytes uncompress key
func (d Driver) PubKeyFromBytes(b []byte) (crypto.PubKey, error) {
	if len(b) != pubkeyBytesLen && len(b) != 33 {
		return nil, errors.New("invalid pub key byte,must be 65 bytes")
	}
	if len(b) == 33 {
		p, err := ethcrypto.DecompressPubkey(b)
		if err != nil {
			return nil, err
		}
		b = ethcrypto.FromECDSAPub(p)
	}
	pubKeyBytes := new([pubkeyBytesLen]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySecp256k1Sha3(*pubKeyBytes), nil
}

//Validate check signature
func (d Driver) Validate(msg, pub, sig []byte) error {
	return crypto.BasicValidation(d, msg, pub, sig)
}

//GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(32))
	//fmt.Println(fmt.Sprintf("GenKey:%x", privKeyBytes))
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return PrivKeySecp256k1Sha3(privKeyBytes), nil
}

//Bytes 字节格式
func (privKey PrivKeySecp256k1Sha3) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, privKey[:])
	return s
}

//Sign 签名 The produced signature is in the [R || S || V] format where V is 0 or 1.
func (privKey PrivKeySecp256k1Sha3) Sign(msg []byte) crypto.Signature {
	priv, err := ethcrypto.ToECDSA(privKey[:])
	if err != nil {
		return nil
	}
	hash := signHash(msg)
	sig, err := ethcrypto.Sign(hash, priv)
	if err != nil {
		panic("Error Sign calculates an ECDSA signature." + err.Error())
	}
	return SignatureSecp256k1Sha3(sig)
}

//PubKey 私钥生成公钥 非压缩 65 bytes 0x04+pub.X+pub.Y
func (privKey PrivKeySecp256k1Sha3) PubKey() crypto.PubKey {
	priv, err := ethcrypto.ToECDSA(privKey[:])
	if nil != err {
		return nil
	}
	//uncompressed pubkey
	var pubSecp256k1 PubKeySecp256k1Sha3
	copy(pubSecp256k1[:], ethcrypto.FromECDSAPub(&priv.PublicKey))
	return pubSecp256k1
}

//Equals 私钥是否相等
func (privKey PrivKeySecp256k1Sha3) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySecp256k1Sha3); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	}
	return false

}

func (privKey PrivKeySecp256k1Sha3) String() string {
	return fmt.Sprintf("PrivKeySecp256k1{*****}")
}

//SignatureSecp256k1 Signature
type SignatureSecp256k1Sha3 []byte

//SignatureS 签名
type SignatureS struct {
	crypto.Signature
}

//Bytes 字节格式
func (sig SignatureSecp256k1Sha3) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

//IsZero 是否是0
func (sig SignatureSecp256k1Sha3) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSecp256k1Sha3) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

//Equals 相等
func (sig SignatureSecp256k1Sha3) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureSecp256k1Sha3); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false

}

// PubKey

//PubKeySecp256k1Sha3 uncompressed pubkey (just the x-cord),
// prefixed with 0x04
type PubKeySecp256k1Sha3 [65]byte

//Bytes 字节格式
func (pubKey PubKeySecp256k1Sha3) Bytes() []byte {
	s := make([]byte, 65)
	copy(s, pubKey[:])
	return s
}

//VerifyBytes 验证字节
func (pubKey PubKeySecp256k1Sha3) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	hash := signHash(msg)
	recoverPub, err := ethcrypto.Ecrecover(hash, sig.Bytes())
	if err != nil {
		return false
	}

	if !bytes.Equal(recoverPub, pubKey[:]) {
		return false
	}

	return ethcrypto.VerifySignature(pubKey[:], hash, sig.Bytes()[:64])

}

func (pubKey PubKeySecp256k1Sha3) String() string {
	return fmt.Sprintf("PubKeySecp256k1{%X}", pubKey[:])
}

//KeyString Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySecp256k1Sha3) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

//Equals 公钥相等
func (pubKey PubKeySecp256k1Sha3) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySecp256k1Sha3); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	}
	return false

}

// signHash is a helper function that calculates a hash for the given message
// that can be safely used to calculate a signature from.
//
// The hash is calulcated as
//   keccak256("\x19Ethereum Signed Message:\n"${message length}${message}).
func signHash(data []byte) []byte {
	return ethcrypto.Keccak256(data)
	//msg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), data)
	//return ethcrypto.Keccak256([]byte(msg))
}

//const
const (
	Name = "secp256k1sha3"
	ID   = 260
)

func init() {
	crypto.Register(Name, &Driver{}, crypto.WithRegOptionTypeID(ID))
}
