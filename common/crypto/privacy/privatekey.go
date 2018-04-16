package privacy

import (
	"bytes"
	"unsafe"
	"gitlab.33.cn/chain33/chain33/common/ed25519"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
)


type PrivKeyPrivacy [PrivateKeyLen]byte

func (privKey PrivKeyPrivacy) Bytes() []byte {
	return privKey[:]
}

func (privKey PrivKeyPrivacy) Sign(msg []byte) Signature {
	privKeyBytes := [64]byte(privKey)
	signatureBytes := ed25519.Sign(&privKeyBytes, msg)
	return SignatureOnetime(*signatureBytes)
}

func (privKey PrivKeyPrivacy) PubKey() PubKey {

	var pubKeyPrivacy PubKeyPrivacy

	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(&privKey.Bytes()[0]))
	addr64 := (*[PrivateKeyLen]byte)(unsafe.Pointer(&privKey.Bytes()[0]))

	var A edwards25519.ExtendedGroupElement
	pubKeyAddr32 := (*[KeyLen32]byte)(unsafe.Pointer(&pubKeyPrivacy))
	edwards25519.GeScalarMultBase(&A, addr32)
	A.ToBytes(pubKeyAddr32)
	copy(addr64[KeyLen32:], pubKeyAddr32[:])

	return pubKeyPrivacy
}

func (privKey PrivKeyPrivacy) Equals(other PrivKey) bool {
	if otherEd, ok := other.(PrivKeyPrivacy); ok {
		return bytes.Equal(privKey[:], otherEd[:])
	} else {
		return false
	}
}

