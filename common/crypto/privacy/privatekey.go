package privacy

import (
	"bytes"
	"unsafe"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/sha3"
	"fmt"
	"gitlab.33.cn/chain33/chain33/common"
)


type PrivKeyPrivacy [PrivateKeyLen]byte

func (privKey PrivKeyPrivacy) Bytes() []byte {
	return privKey[:]
}

func (privKey PrivKeyPrivacy) Sign(msg []byte) Signature {
	fmt.Printf("~~~~~~~~~~~~~~privKey is:%X\n", privKey[:])

	temp := new([64]byte)
	randomScalar := new([32]byte)
    copy(temp[:], CRandBytes(64))
	edwards25519.ScReduce(randomScalar, temp)

	////////////////////
	randomScalarStub, _ := common.Hex2Bytes("FE7B1D8218D0B02C402BCA5AB2F6B6726C662E5CADA915D0D72323CA2F60D403")
	copy(randomScalar[:], randomScalarStub)
	fmt.Printf("~~~~~~~~~~~~~~randomScalar is:%X\n", randomScalar[:])
	////////////////////

	var sigcommdata sigcommArray
	sigcommPtr := (*sigcomm)(unsafe.Pointer(&sigcommdata))
	copy(sigcommPtr.pubkey[:], privKey.PubKey().Bytes())
	hash := sha3.Sum256(msg)
	copy(sigcommPtr.hash[:], hash[:])

	var K edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&K, randomScalar)
	K.ToBytes((*[KeyLen32]byte)(unsafe.Pointer(&sigcommPtr.comm[0])))
	fmt.Printf("~~~~~~~~~~~~~~sigcommPtr.comm is:%x\n", sigcommPtr.comm[:])

	var sigOnetime SignatureOnetime
	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(&sigOnetime))
	hash2scalar(sigcommdata[:], addr32)
	fmt.Printf("~~~~~~~~~~~~~~hash2scalar hash is:%X\n", addr32[:])

	addr32Latter := (*[KeyLen32]byte)(unsafe.Pointer(&sigOnetime[KeyLen32]))
	addr32Priv := (*[KeyLen32]byte)(unsafe.Pointer(&privKey))
	fmt.Printf("~~~~~~~~~~~~~~The input privKey :%x\n", addr32Priv[:])
	fmt.Printf("~~~~~~~~~~~~~~The randomScalar :%x\n", randomScalar[:])
	edwards25519.ScMulSub(addr32Latter, addr32, addr32Priv, randomScalar)

	fmt.Printf("~~~~~~~~~~~~~~The new generated signature :%x\n", sigOnetime[:])

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

