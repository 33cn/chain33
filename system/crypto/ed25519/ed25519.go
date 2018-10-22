package ed25519

import (
	"bytes"
	"errors"
	"fmt"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519"
)

type Driver struct{}

// Crypto
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], crypto.CRandBytes(32))
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes), nil
}

func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != 64 {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], b[:32])
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes), nil
}

func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != 32 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([32]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeyEd25519(*pubKeyBytes), nil
}

func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	sigBytes := new([64]byte)
	copy(sigBytes[:], b[:])
	return SignatureEd25519(*sigBytes), nil
}

// PrivKey
type PrivKeyEd25519 [64]byte

func (privKey PrivKeyEd25519) Bytes() []byte {
	s := make([]byte, 64)
	copy(s, privKey[:])
	return s
}

func (privKey PrivKeyEd25519) Sign(msg []byte) crypto.Signature {
	privKeyBytes := [64]byte(privKey)
	signatureBytes := ed25519.Sign(&privKeyBytes, msg)
	return SignatureEd25519(*signatureBytes)
}

func (privKey PrivKeyEd25519) PubKey() crypto.PubKey {
	privKeyBytes := [64]byte(privKey)
	return PubKeyEd25519(*ed25519.MakePublicKey(&privKeyBytes))
}

func (privKey PrivKeyEd25519) Equals(other crypto.PrivKey) bool {
	if otherEd, ok := other.(PrivKeyEd25519); ok {
		return bytes.Equal(privKey[:], otherEd[:])
	} else {
		return false
	}
}

// PubKey
type PubKeyEd25519 [32]byte

func (pubKey PubKeyEd25519) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, pubKey[:])
	return s
}

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

func (pubKey PubKeyEd25519) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeyEd25519) Equals(other crypto.PubKey) bool {
	if otherEd, ok := other.(PubKeyEd25519); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	} else {
		return false
	}
}

// Signature
type SignatureEd25519 [64]byte

type SignatureS struct {
	crypto.Signature
}

func (sig SignatureEd25519) Bytes() []byte {
	s := make([]byte, 64)
	copy(s, sig[:])
	return s
}

func (sig SignatureEd25519) IsZero() bool { return len(sig) == 0 }

func (sig SignatureEd25519) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)
}

func (sig SignatureEd25519) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(SignatureEd25519); ok {
		return bytes.Equal(sig[:], otherEd[:])
	} else {
		return false
	}
}

const Name = "ed25519"
const ID = 2

func init() {
	crypto.Register(Name, &Driver{})
	crypto.RegisterType(Name, ID)
}
