package privacy

import (
	"bytes"
	"fmt"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	"gitlab.33.cn/chain33/chain33/common/crypto/sha3"
	"unsafe"
	"crypto/subtle"
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

	var ege edwards25519.ExtendedGroupElement
	if !ege.FromBytes(&pubKeyBytes) {
		return false
	}

	sigAddr32a := (*[KeyLen32]byte)(unsafe.Pointer(&sigBytes[0]))
	sigAddr32b := (*[KeyLen32]byte)(unsafe.Pointer(&sigBytes[32]))
	if !edwards25519.ScCheck(sigAddr32a) || !edwards25519.ScCheck(sigAddr32b) {
		return false
	}

	var sigcommdata sigcommArray
	sigcommPtr := (*sigcomm)(unsafe.Pointer(&sigcommdata))
	copy(sigcommPtr.pubkey[:], pubKey.Bytes())
	hash := sha3.Sum256(msg)
	copy(sigcommPtr.hash[:], hash[:])

	var rge edwards25519.ProjectiveGroupElement
	edwards25519.GeDoubleScalarMultVartime(&rge, sigAddr32a, &ege, sigAddr32b)
	rge.ToBytes((*[KeyLen32]byte)(unsafe.Pointer(&sigcommPtr.comm[0])))

	out32 := new([32]byte)
	hash2scalar(sigcommdata[:], out32)
	return subtle.ConstantTimeCompare(sigAddr32a[:], out32[:]) == 1
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
