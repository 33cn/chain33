package privacy

import (
	"fmt"
	"bytes"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519"
)

type PubKeyPrivacy [PublicKeyLen]byte

func (pubKey PubKeyPrivacy) Bytes() []byte {
	return pubKey[:]
}

func (pubKey PubKeyPrivacy) VerifyBytes(msg []byte, sig_ Signature) bool {
	// unwrap if needed
	if wrap, ok := sig_.(SignatureS); ok {
		sig_ = wrap.Signature
	}
	// make sure we use the same algorithm to sign
	sig, ok := sig_.(SignatureOnetime)
	if !ok {
		return false
	}
	pubKeyBytes := [32]byte(pubKey)
	sigBytes := [64]byte(sig)
	return ed25519.Verify(&pubKeyBytes, msg, &sigBytes)
}

func (pubKey PubKeyPrivacy) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeyPrivacy) Equals(other PubKey) bool {
	if otherEd, ok := other.(PubKeyPrivacy); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	} else {
		return false
	}
}
