package sm2

import (
	"bytes"
	"encoding/hex"
	"fmt"

	. "code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/common/crypto/gosm2"
)

type Sm2Driver struct{}

const (
	TypeSm2 = byte(0x03)
	NameSm2 = "sm2"
)

// Crypto
func (d Sm2Driver) GenKey() (PrivKey, error) {
	priv, err := gosm2.NewKey()
	if err != nil {
		return nil, err
	}

	return PrivKeySm2{priv: priv}, nil
}

func (d Sm2Driver) PrivKeyFromBytes(b []byte) (PrivKey, error) {
	priv, err := gosm2.DERDecodePrivateKey(b)
	if err != nil {
		return nil, err
	}

	return PrivKeySm2{priv: priv}, nil
}

func (d Sm2Driver) PubKeyFromBytes(b []byte) (PubKey, error) {
	var buf []byte
	if len(b) == 64 {
		buf = make([]byte, 65)
		preFix, _ := hex.DecodeString("04")
		copy(buf, preFix)
		copy(buf[len(preFix):65], b)
	} else {
		buf = b
	}

	pub, err := gosm2.OStrDecodePublicKey(buf)
	if err != nil {
		return nil, err
	}

	return PubKeySm2{pub: pub}, nil
}

func (d Sm2Driver) SignatureFromBytes(b []byte) (Signature, error) {
	var sigBytes []byte
	if len(b) == 64 {
		sig2, err := gosm2.Sigbin2Der(b)
		if err != nil {
			return nil, err
		}
		sigBytes = sig2
	} else {
		sigBytes = b
	}

	return SignatureSm2(sigBytes), nil
}

// PrivKey
type PrivKeySm2 struct {
	priv *gosm2.PriveKey
}

func (privKey PrivKeySm2) Bytes() []byte {
	bytez, err := privKey.priv.DEREncode()
	if err != nil {
		return nil
	}
	return bytez
}

func (privKey PrivKeySm2) Sign(msg []byte) Signature {
	hash, err := gosm2.Hash(privKey.priv.PublicKey, msg)
	if err != nil {
		panic(err)
	}

	bytez, err := gosm2.Sign(privKey.priv, hash)
	if err != nil {
		panic(err)
	}

	return SignatureSm2(bytez)
}

func (privKey PrivKeySm2) PubKey() PubKey {
	return PubKeySm2{pub: privKey.priv.PublicKey}
}

func (privKey PrivKeySm2) Equals(other PrivKey) bool {
	if otherPriv, ok := other.(PrivKeySm2); ok {
		return bytes.Equal(privKey.Bytes(), otherPriv.Bytes())
	} else {
		return false
	}
}

// PubKey
type PubKeySm2 struct {
	pub *gosm2.PublicKey
}

func (pubKey PubKeySm2) Bytes() []byte {
	bytez, err := pubKey.pub.OStrEncode()
	if err != nil {
		return nil
	}
	return bytez
}

func (pubKey PubKeySm2) VerifyBytes(msg []byte, sig Signature) bool {
	hash, err := gosm2.Hash(pubKey.pub, msg)
	if err != nil {
		panic(err)
	}

	return gosm2.Verify(pubKey.pub, hash, sig.Bytes()) == nil
}

func (pubKey PubKeySm2) KeyString() string {
	return pubKey.pub.EncodeString()
}

func (pubKey PubKeySm2) Equals(other PubKey) bool {
	if otherPub, ok := other.(PubKeySm2); ok {
		return bytes.Equal(pubKey.Bytes(), otherPub.Bytes())
	} else {
		return false
	}
}

// Signature
type SignatureSm2 []byte

func (sig SignatureSm2) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig)
	return s
}

func (sig SignatureSm2) IsZero() bool {
	return len(sig) == 0
}

func (sig SignatureSm2) String() string {
	return fmt.Sprintf("/%X.../", sig.Bytes())
}

func (sig SignatureSm2) Equals(other Signature) bool {
	if otherEd, ok := other.(SignatureSm2); ok {
		return bytes.Equal(sig, otherEd)
	} else {
		return false
	}
}

func init() {
	Register(NameSm2, &Sm2Driver{})
}
