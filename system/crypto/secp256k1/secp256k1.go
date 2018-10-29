package secp256k1

import (
	"bytes"
	"errors"
	"fmt"

	secp256k1 "github.com/btcsuite/btcd/btcec"
	"gitlab.33.cn/chain33/chain33/common/crypto"
)

type Driver struct{}

// Ctypto
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(32))
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return PrivKeySecp256k1(privKeyBytes), nil
}

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

func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != 33 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([33]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySecp256k1(*pubKeyBytes), nil
}

func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	return SignatureSecp256k1(b), nil
}

// PrivKey
type PrivKeySecp256k1 [32]byte

func (privKey PrivKeySecp256k1) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, privKey[:])
	return s
}

func (privKey PrivKeySecp256k1) Sign(msg []byte) crypto.Signature {
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey[:])
	sig, err := priv.Sign(crypto.Sha256(msg))
	if err != nil {
		panic("Error signing secp256k1" + err.Error())
	}
	return SignatureSecp256k1(sig.Serialize())
}

func (privKey PrivKeySecp256k1) PubKey() crypto.PubKey {
	_, pub := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey[:])
	var pubSecp256k1 PubKeySecp256k1
	copy(pubSecp256k1[:], pub.SerializeCompressed())
	return pubSecp256k1
}

func (privKey PrivKeySecp256k1) Equals(other crypto.PrivKey) bool {
	if otherSecp, ok := other.(PrivKeySecp256k1); ok {
		return bytes.Equal(privKey[:], otherSecp[:])
	} else {
		return false
	}
}

func (privKey PrivKeySecp256k1) String() string {
	return fmt.Sprintf("PrivKeySecp256k1{*****}")
}

// PubKey

// Compressed pubkey (just the x-cord),
// prefixed with 0x02 or 0x03, depending on the y-cord.
type PubKeySecp256k1 [33]byte

func (pubKey PubKeySecp256k1) Bytes() []byte {
	s := make([]byte, 33)
	copy(s, pubKey[:])
	return s
}

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

// Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySecp256k1) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeySecp256k1) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(PubKeySecp256k1); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	} else {
		return false
	}
}

// Signature
type SignatureSecp256k1 []byte

type SignatureS struct {
	crypto.Signature
}

func (sig SignatureSecp256k1) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

func (sig SignatureSecp256k1) IsZero() bool { return len(sig) == 0 }

func (sig SignatureSecp256k1) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

func (sig SignatureSecp256k1) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureSecp256k1); ok {
		return bytes.Equal(sig[:], otherEd[:])
	} else {
		return false
	}
}

const Name = "secp256k1"
const ID = 1

func init() {
	crypto.Register(Name, &Driver{})
	crypto.RegisterType(Name, ID)
}
