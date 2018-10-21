package ecdsa

import (
	"bytes"
	"errors"
	"fmt"

	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/asn1"
	"math/big"

	"gitlab.33.cn/chain33/chain33/common/crypto"
)

const (
	ECDSA_RPIVATEKEY_LENGTH = 32
	ECDSA_PUBLICKEY_LENGTH  = 65
)

type Driver struct{}

// Ctypto
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [ECDSA_RPIVATEKEY_LENGTH]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(ECDSA_RPIVATEKEY_LENGTH))
	priv, _ := privKeyFromBytes(elliptic.P256(), privKeyBytes[:])
	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeyECDSA(privKeyBytes), nil
}

func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != ECDSA_RPIVATEKEY_LENGTH {
		return nil, errors.New("invalid priv key byte")
	}

	privKeyBytes := new([ECDSA_RPIVATEKEY_LENGTH]byte)
	copy(privKeyBytes[:], b[:ECDSA_RPIVATEKEY_LENGTH])
	priv, _ := privKeyFromBytes(elliptic.P256(), privKeyBytes[:])

	copy(privKeyBytes[:], SerializePrivateKey(priv))
	return PrivKeyECDSA(*privKeyBytes), nil
}

func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != ECDSA_PUBLICKEY_LENGTH {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([ECDSA_PUBLICKEY_LENGTH]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeyECDSA(*pubKeyBytes), nil
}

func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	var certSignature crypto.CertSignature
	_, err = asn1.Unmarshal(b, &certSignature)
	if err != nil {
		return SignatureECDSA(b), nil
	}

	if len(certSignature.Cert) == 0 {
		return SignatureECDSA(b), nil
	}

	return SignatureECDSA(certSignature.Signature), nil
}

// PrivKey
type PrivKeyECDSA [ECDSA_RPIVATEKEY_LENGTH]byte

func (privKey PrivKeyECDSA) Bytes() []byte {
	s := make([]byte, ECDSA_RPIVATEKEY_LENGTH)
	copy(s, privKey[:])
	return s
}

func (privKey PrivKeyECDSA) Sign(msg []byte) crypto.Signature {
	priv, pub := privKeyFromBytes(elliptic.P256(), privKey[:])
	r, s, err := ecdsa.Sign(rand.Reader, priv, crypto.Sha256(msg))
	if err != nil {
		return nil
	}

	s = ToLowS(pub, s)
	ecdsaSigByte, _ := MarshalECDSASignature(r, s)
	return SignatureECDSA(ecdsaSigByte)
}

func (privKey PrivKeyECDSA) PubKey() crypto.PubKey {
	_, pub := privKeyFromBytes(elliptic.P256(), privKey[:])
	var pubECDSA PubKeyECDSA
	copy(pubECDSA[:], SerializePublicKey(pub))
	return pubECDSA
}

func (privKey PrivKeyECDSA) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeyECDSA); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	}

	return false
}

func (privKey PrivKeyECDSA) String() string {
	return fmt.Sprintf("PrivKeyECDSA{*****}")
}

// PubKey
// prefixed with 0x02 or 0x03, depending on the y-cord.
type PubKeyECDSA [ECDSA_PUBLICKEY_LENGTH]byte

func (pubKey PubKeyECDSA) Bytes() []byte {
	s := make([]byte, ECDSA_PUBLICKEY_LENGTH)
	copy(s, pubKey[:])
	return s
}

func (pubKey PubKeyECDSA) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	// unwrap if needed
	if wrap, ok := sig.(SignatureS); ok {
		sig = wrap.Signature
	}
	// and assert same algorithm to sign and verify
	sigECDSA, ok := sig.(SignatureECDSA)
	if !ok {
		return false
	}

	pub, err := parsePubKey(pubKey[:], elliptic.P256())
	if err != nil {
		return false
	}

	r, s, err := UnmarshalECDSASignature(sigECDSA)
	if err != nil {
		return false
	}

	lowS := IsLowS(s)
	if !lowS {
		return false
	}
	return ecdsa.Verify(pub, crypto.Sha256(msg), r, s)
}

func (pubKey PubKeyECDSA) String() string {
	return fmt.Sprintf("PubKeyECDSA{%X}", pubKey[:])
}

// Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeyECDSA) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeyECDSA) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeyECDSA); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	} else {
		return false
	}
}

type ECDSASignature struct {
	R, S *big.Int
}

// Signature
type SignatureECDSA []byte

type SignatureS struct {
	crypto.Signature
}

func (sig SignatureECDSA) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

func (sig SignatureECDSA) IsZero() bool { return len(sig) == 0 }

func (sig SignatureECDSA) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

func (sig SignatureECDSA) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureECDSA); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false
}

const Name = "auth_ecdsa"
const ID = 257

func init() {
	crypto.Register(Name, &Driver{})
	crypto.RegisterType(Name, ID)
}

func privKeyFromBytes(curve elliptic.Curve, pk []byte) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	x, y := curve.ScalarBaseMult(pk)

	priv := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: curve,
			X:     x,
			Y:     y,
		},
		D: new(big.Int).SetBytes(pk),
	}

	return priv, &priv.PublicKey
}
