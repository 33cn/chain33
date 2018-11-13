// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

const (
	SM2_RPIVATEKEY_LENGTH = 32
	SM2_PUBLICKEY_LENGTH  = 65
)

type Driver struct{}

func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [SM2_RPIVATEKEY_LENGTH]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(SM2_RPIVATEKEY_LENGTH))
	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKeyBytes[:])
	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeySM2(privKeyBytes), nil
}

func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != SM2_RPIVATEKEY_LENGTH {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([SM2_RPIVATEKEY_LENGTH]byte)
	copy(privKeyBytes[:], b[:SM2_RPIVATEKEY_LENGTH])

	priv, _ := privKeyFromBytes(sm2.P256Sm2(), privKeyBytes[:])

	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeySM2(*privKeyBytes), nil
}

func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != SM2_PUBLICKEY_LENGTH {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([SM2_PUBLICKEY_LENGTH]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySM2(*pubKeyBytes), nil
}

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

type PrivKeySM2 [SM2_RPIVATEKEY_LENGTH]byte

func (privKey PrivKeySM2) Bytes() []byte {
	s := make([]byte, SM2_RPIVATEKEY_LENGTH)
	copy(s, privKey[:])
	return s
}

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

func (privKey PrivKeySM2) PubKey() crypto.PubKey {
	_, pub := privKeyFromBytes(sm2.P256Sm2(), privKey[:])
	var pubSM2 PubKeySM2
	copy(pubSM2[:], SerializePublicKey(pub))
	return pubSM2
}

func (privKey PrivKeySM2) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySM2); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	}

	return false
}

func (privKey PrivKeySM2) String() string {
	return fmt.Sprintf("PrivKeySM2{*****}")
}

type PubKeySM2 [SM2_PUBLICKEY_LENGTH]byte

func (pubKey PubKeySM2) Bytes() []byte {
	s := make([]byte, SM2_PUBLICKEY_LENGTH)
	copy(s, pubKey[:])
	return s
}

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

// Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySM2) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeySM2) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySM2); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	} else {
		return false
	}
}

type SignatureSM2 []byte

type SignatureS struct {
	crypto.Signature
}

func (sig SignatureSM2) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

func (sig SignatureSM2) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSM2) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

func (sig SignatureSM2) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureSM2); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false
}

const Name = "sm2"
const ID = 3

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
