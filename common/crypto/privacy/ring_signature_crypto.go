/*
基于框架中Crypto接口，实现签名、验证的处理
*/
package privacy

import (
	"errors"
	"unsafe"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	crypto.Register(crypto.SignNameRing, &RingSignED25519{})
}

type RingSignature struct {
	sign types.RingSignature
}

func (r *RingSignature) Bytes() []byte {
	return types.Encode(&r.sign)
}

func (r *RingSignature) IsZero() bool {
	return len(r.sign.GetItems()) == 0
}

func (r *RingSignature) String() string {
	return r.sign.String()
}

func (r *RingSignature) Equals(other crypto.Signature) bool {
	return false
}

type RingSignED25519 struct {
}

func (r *RingSignED25519) GenKey() (crypto.PrivKey, error) {
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

func (r *RingSignED25519) PrivKeyFromBytes(b []byte) (crypto.PrivKey, error) {
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

func (r *RingSignED25519) PubKeyFromBytes(b []byte) (crypto.PubKey, error) {
	if len(b) != 32 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([32]byte)
	copy(pubKeyBytes[:], b[:])
	return PubKeyPrivacy(*pubKeyBytes), nil
}

func (r *RingSignED25519) SignatureFromBytes(b []byte) (crypto.Signature, error) {
	var sign RingSignature
	if err := types.Decode(b, &sign.sign); err != nil {
		return nil, err
	}
	return &sign, nil
}
