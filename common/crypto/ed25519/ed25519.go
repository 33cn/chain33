package ed25519

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/tendermint/ed25519"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
)

type Ed25519Driver struct{}

const (
	TypeEd25519 = byte(0x01)
	NameEd25519 = "ed25519"
)

// Crypto
func (d Ed25519Driver) GenKey() (PrivKey, error) {
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], CRandBytes(32))
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes), nil
}

func (d Ed25519Driver) PrivKeyFromBytes(b []byte) (privKey PrivKey, err error) {
	if len(b) != 64 {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([64]byte)
	copy(privKeyBytes[:32], b[:32])
	ed25519.MakePublicKey(privKeyBytes)
	return PrivKeyEd25519(*privKeyBytes), nil
}

func (d Ed25519Driver) PubKeyFromBytes(b []byte) (pubKey PubKey, err error) {
	if len(b) != 32 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([32]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeyEd25519(*pubKeyBytes), nil
}

func (d Ed25519Driver) SignatureFromBytes(b []byte) (sig Signature, err error) {
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

func (privKey PrivKeyEd25519) Sign(msg []byte) Signature {
	privKeyBytes := [64]byte(privKey)
	signatureBytes := ed25519.Sign(&privKeyBytes, msg)
	return SignatureEd25519(*signatureBytes)
}

func (privKey PrivKeyEd25519) PubKey() PubKey {
	privKeyBytes := [64]byte(privKey)
	return PubKeyEd25519(*ed25519.MakePublicKey(&privKeyBytes))
}

func (privKey PrivKeyEd25519) Equals(other PrivKey) bool {
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

func (pubKey PubKeyEd25519) VerifyBytes(msg []byte, sig_ Signature) bool {
	// unwrap if needed
	if wrap, ok := sig_.(SignatureS); ok {
		sig_ = wrap.Signature
	}
	// make sure we use the same algorithm to sign
	sig, ok := sig_.(SignatureEd25519)
	if !ok {
		return false
	}
	pubKeyBytes := [32]byte(pubKey)
	sigBytes := [64]byte(sig)
	return ed25519.Verify(&pubKeyBytes, msg, &sigBytes)
}

func (pubKey PubKeyEd25519) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeyEd25519) Equals(other PubKey) bool {
	if otherEd, ok := other.(PubKeyEd25519); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	} else {
		return false
	}
}

// Signature
type SignatureEd25519 [64]byte

type SignatureS struct {
	Signature
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

func (sig SignatureEd25519) Equals(other Signature) bool {
	if otherEd, ok := other.(SignatureEd25519); ok {
		return bytes.Equal(sig[:], otherEd[:])
	} else {
		return false
	}
}

func init() {
	Register(NameEd25519, &Ed25519Driver{})
}
