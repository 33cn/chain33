package privacy

import (
	"errors"
	"unsafe"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
)

type OneTimeEd25519 struct{}

func init() {
	crypto.Register(privacytypes.SignNameOnetimeED25519, &OneTimeEd25519{})
}

func (onetime *OneTimeEd25519) GenKey() (crypto.PrivKey, error) {
	privKeyPrivacyPtr := &PrivKeyPrivacy{}
	pubKeyPrivacyPtr := &PubKeyPrivacy{}
	copy(privKeyPrivacyPtr[:PrivateKeyLen], crypto.CRandBytes(PrivateKeyLen))

	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(privKeyPrivacyPtr))
	addr64 := (*[PrivateKeyLen]byte)(unsafe.Pointer(privKeyPrivacyPtr))
	edwards25519.ScReduce(addr32, addr64)

	//to generate the publickey
	var A edwards25519.ExtendedGroupElement
	pubKeyAddr32 := (*[KeyLen32]byte)(unsafe.Pointer(pubKeyPrivacyPtr))
	edwards25519.GeScalarMultBase(&A, addr32)
	A.ToBytes(pubKeyAddr32)
	copy(addr64[KeyLen32:], pubKeyAddr32[:])

	return *privKeyPrivacyPtr, nil
}

func (onetime *OneTimeEd25519) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != 64 {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([PrivateKeyLen]byte)
	pubKeyBytes := new([PublicKeyLen]byte)
	copy(privKeyBytes[:KeyLen32], b[:KeyLen32])

	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(privKeyBytes))
	addr64 := (*[PrivateKeyLen]byte)(unsafe.Pointer(privKeyBytes))

	//to generate the publickey
	var A edwards25519.ExtendedGroupElement
	pubKeyAddr32 := (*[KeyLen32]byte)(unsafe.Pointer(pubKeyBytes))
	edwards25519.GeScalarMultBase(&A, addr32)
	A.ToBytes(pubKeyAddr32)
	copy(addr64[KeyLen32:], pubKeyAddr32[:])

	return PrivKeyPrivacy(*privKeyBytes), nil
}

func (onetime *OneTimeEd25519) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != 32 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([32]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeyPrivacy(*pubKeyBytes), nil
}

func (onetime *OneTimeEd25519) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	sigBytes := new([64]byte)
	copy(sigBytes[:], b[:])
	return SignatureOnetime(*sigBytes), nil
}
