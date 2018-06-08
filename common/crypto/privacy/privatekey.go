package privacy

import (
	"bytes"
	"unsafe"

	. "gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/sha3"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
)

type PrivKeyPrivacy [PrivateKeyLen]byte

func (privKey PrivKeyPrivacy) Bytes() []byte {
	return privKey[:]
}

func (privKey PrivKeyPrivacy) Sign(msg []byte) Signature {

	temp := new([64]byte)
	randomScalar := new([32]byte)
	copy(temp[:], CRandBytes(64))
	edwards25519.ScReduce(randomScalar, temp)

	var sigcommdata sigcommArray
	sigcommPtr := (*sigcomm)(unsafe.Pointer(&sigcommdata))
	copy(sigcommPtr.pubkey[:], privKey.PubKey().Bytes())
	hash := sha3.Sum256(msg)
	copy(sigcommPtr.hash[:], hash[:])

	var K edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&K, randomScalar)
	K.ToBytes((*[KeyLen32]byte)(unsafe.Pointer(&sigcommPtr.comm[0])))

	var sigOnetime SignatureOnetime
	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(&sigOnetime))
	hash2scalar(sigcommdata[:], addr32)

	addr32Latter := (*[KeyLen32]byte)(unsafe.Pointer(&sigOnetime[KeyLen32]))
	addr32Priv := (*[KeyLen32]byte)(unsafe.Pointer(&privKey))
	edwards25519.ScMulSub(addr32Latter, addr32, addr32Priv, randomScalar)

	return sigOnetime
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
