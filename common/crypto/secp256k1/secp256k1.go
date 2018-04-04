package secp256k1

import (
	"bytes"
	"errors"
	"fmt"

	secp256k1 "github.com/btcsuite/btcd/btcec"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
)

type Secp256k1Driver struct{}

const (
	TypeSecp256k1 = byte(0x02)
	NameSecp256k1 = "secp256k1"
)

// Ctypto
func (d Secp256k1Driver) GenKey() (PrivKey, error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], CRandBytes(32))
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return PrivKeySecp256k1(privKeyBytes), nil
}

func (d Secp256k1Driver) PrivKeyFromBytes(b []byte) (privKey PrivKey, err error) {
	if len(b) != 32 {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([32]byte)
	copy(privKeyBytes[:], b[:32])
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return PrivKeySecp256k1(*privKeyBytes), nil
}

func (d Secp256k1Driver) PubKeyFromBytes(b []byte) (pubKey PubKey, err error) {
	if len(b) != 33 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([33]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeySecp256k1(*pubKeyBytes), nil
}

func (d Secp256k1Driver) SignatureFromBytes(b []byte) (sig Signature, err error) {
	return SignatureSecp256k1(b), nil
}

// PrivKey
type PrivKeySecp256k1 [32]byte

func (privKey PrivKeySecp256k1) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, privKey[:])
	return s
}

func (privKey PrivKeySecp256k1) Sign(msg []byte) Signature {
	priv__, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey[:])
	sig__, err := priv__.Sign(Sha256(msg))
	if err != nil {
		panic("Error signing secp256k1" + err.Error())
	}
	return SignatureSecp256k1(sig__.Serialize())
}

func (privKey PrivKeySecp256k1) PubKey() PubKey {
	_, pub__ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKey[:])
	var pub PubKeySecp256k1
	copy(pub[:], pub__.SerializeCompressed())
	return pub
}

func (privKey PrivKeySecp256k1) Equals(other PrivKey) bool {
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

func (pubKey PubKeySecp256k1) VerifyBytes(msg []byte, sig_ Signature) bool {
	// unwrap if needed
	if wrap, ok := sig_.(SignatureS); ok {
		sig_ = wrap.Signature
	}
	// and assert same algorithm to sign and verify
	sig, ok := sig_.(SignatureSecp256k1)
	if !ok {
		return false
	}

	pub__, err := secp256k1.ParsePubKey(pubKey[:], secp256k1.S256())
	if err != nil {
		return false
	}
	sig__, err := secp256k1.ParseDERSignature(sig[:], secp256k1.S256())
	if err != nil {
		return false
	}
	return sig__.Verify(Sha256(msg), pub__)
}

func (pubKey PubKeySecp256k1) String() string {
	return fmt.Sprintf("PubKeySecp256k1{%X}", pubKey[:])
}

// Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey PubKeySecp256k1) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeySecp256k1) Equals(other PubKey) bool {
	if otherSecp, ok := other.(PubKeySecp256k1); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	} else {
		return false
	}
}

// Signature
type SignatureSecp256k1 []byte

type SignatureS struct {
	Signature
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

func (sig SignatureSecp256k1) Equals(other Signature) bool {
	if otherEd, ok := other.(SignatureSecp256k1); ok {
		return bytes.Equal(sig[:], otherEd[:])
	} else {
		return false
	}
}

func init() {
	Register(NameSecp256k1, &Secp256k1Driver{})
}
