package privacy

import (
	"bytes"
	"errors"
	"io"
	"unsafe"

	sccrypto "github.com/NebulousLabs/Sia/crypto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/crypto/sha3"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"

	//"gitlab.33.cn/chain33/chain33/types"
	"fmt"
)

const (
	PublicKeyLen  = 32
	PrivateKeyLen = 64
	KeyLen32      = 32
)

type Privacy struct {
	ViewPubkey   PubKeyPrivacy
	ViewPrivKey  PrivKeyPrivacy
	SpendPubkey  PubKeyPrivacy
	SpendPrivKey PrivKeyPrivacy
}

type EllipticCurvePoint [32]byte
type sigcomm struct {
	hash   [32]byte
	pubkey EllipticCurvePoint
	comm   EllipticCurvePoint
}

//
type sigcommArray [32 * 3]byte
type KeyImage [32]byte

var (
	ErrViewPub       = errors.New("ErrViewPub")
	ErrSpendPub      = errors.New("ErrSpendPub")
	ErrViewSecret    = errors.New("ErrViewSecret")
	ErrSpendSecret   = errors.New("ErrSpendSecret")
	ErrNullRandInput = errors.New("ErrNullRandInput")
)

var privacylog = log.New("module", "crypto.privacy")

//////////////
func NewPrivacy() *Privacy {
	privacy := &Privacy{}
	GenerateKeyPair(&privacy.SpendPrivKey, &privacy.SpendPubkey)
	GenerateKeyPair(&privacy.ViewPrivKey, &privacy.ViewPubkey)

	return privacy
}

func NewPrivacyWithPrivKey(privKey *[KeyLen32]byte) (privacy *Privacy, err error) {
	privacylog.Info("NewPrivacyWithPrivKey", "input prikey", common.Bytes2Hex(privKey[:]))
	hash := sccrypto.HashAll(*privKey)
	privacy = &Privacy{}

	if err = generateKeyPairWithPrivKey((*[KeyLen32]byte)(unsafe.Pointer(&hash[0])), &privacy.SpendPrivKey, &privacy.SpendPubkey); err != nil {
		return nil, err
	}

	hashViewPriv := sccrypto.HashAll(privacy.SpendPrivKey[0:KeyLen32])
	if err = generateKeyPairWithPrivKey((*[KeyLen32]byte)(unsafe.Pointer(&hashViewPriv[0])), &privacy.ViewPrivKey, &privacy.ViewPubkey); err != nil {
		return nil, err
	}
	privacylog.Info("NewPrivacyWithPrivKey", "the new privacy created with viewpub", common.Bytes2Hex(privacy.ViewPubkey[:]))
	privacylog.Info("NewPrivacyWithPrivKey", "the new privacy created with spendpub", common.Bytes2Hex(privacy.SpendPubkey[:]))

	return privacy, nil
}

//(A, B) => Hs(rA)G + B, rG=>R
//func GenerateOneTimeAddr(viewPub, spendPub, skAddr32 *[32]byte, outputIndex int64) (pubkeyOnetime, RtxPublicKey *[32]byte, errInfo error) {
func GenerateOneTimeAddr(viewPub, spendPub, skAddr32 *[32]byte, outputIndex int64) (pubkeyOnetime *[32]byte, errInfo error) {

	//to calculate rA
	var point edwards25519.ExtendedGroupElement
	if res := point.FromBytes(viewPub); !res {
		return nil, ErrViewPub
	}
	//skAddr32 := (*[KeyLen32]byte)(unsafe.Pointer(sk))
	if !edwards25519.ScCheck(skAddr32) {
		privacylog.Error("xxx GenerateOneTimeAddr Fail to do edwards25519.ScCheck with sk \n")
		return nil, ErrViewSecret
	}
	var point2 edwards25519.ProjectiveGroupElement
	zeroValue := &[32]byte{}
	edwards25519.GeDoubleScalarMultVartime(&point2, skAddr32, &point, zeroValue)

	var point3 edwards25519.CompletedGroupElement
	mul8(&point3, &point2)
	point3.ToProjective(&point2)
	rA := new([32]byte)
	point2.ToBytes(rA)

	//to calculate Hs(rA)G + B
	var B edwards25519.ExtendedGroupElement //A
	if res := B.FromBytes(spendPub); !res {
		return nil, ErrSpendPub
	}
	//Hs(rA)
	Hs_rA := derivation2scalar(rA, outputIndex)

	var A edwards25519.ExtendedGroupElement
	edwards25519.GeScalarMultBase(&A, Hs_rA)
	//A.ToBytes(publicKey)
	var cachedA edwards25519.CachedGroupElement
	//Hs(rA)G
	A.ToCached(&cachedA)
	var point4 edwards25519.CompletedGroupElement
	//Hs(rA)G + B
	edwards25519.GeAdd(&point4, &B, &cachedA)
	var point5 edwards25519.ProjectiveGroupElement
	point4.ToProjective(&point5)
	var onetimePubKey [32]byte

	point5.ToBytes(&onetimePubKey)
	pubkeyOnetime = &onetimePubKey
	return
}

//calculate Hs(aR) + b
func RecoverOnetimePriKey(R []byte, viewSecretKey, spendSecretKey PrivKey, outputIndex int64) (PrivKey, error) {
	var viewSecAddr, spendSecAddr, RtxPubAddr *[32]byte
	viewSecAddr = (*[32]byte)(unsafe.Pointer(&viewSecretKey.Bytes()[0]))
	spendSecAddr = (*[32]byte)(unsafe.Pointer(&spendSecretKey.Bytes()[0]))
	RtxPubAddr = (*[32]byte)(unsafe.Pointer(&R[0]))
	//1st to calculate aR
	var point edwards25519.ExtendedGroupElement
	if res := point.FromBytes(RtxPubAddr); !res {
		privacylog.Error("RecoverOnetimePriKey Fail to do get point.FromBytes with viewSecAddr \n")
		return nil, ErrViewSecret
	}

	if !edwards25519.ScCheck(viewSecAddr) {
		privacylog.Error("xxx RecoverOnetimePriKey Fail to do edwards25519.ScCheck with viewSecAddr \n")
		return nil, ErrViewSecret
	}

	var point2 edwards25519.ProjectiveGroupElement
	zeroValue := &[32]byte{}
	//TODO,编写新的函数只用于计算aR
	edwards25519.GeDoubleScalarMultVartime(&point2, viewSecAddr, &point, zeroValue)
	var point3 edwards25519.CompletedGroupElement
	mul8(&point3, &point2)
	point3.ToProjective(&point2)
	aR := new([32]byte)
	point2.ToBytes(aR)

	if !edwards25519.ScCheck(spendSecAddr) {
		privacylog.Error("xxx RecoverOnetimePriKey Fail to do edwards25519.ScCheck with spendSecAddr \n")
		return nil, ErrViewSecret
	}

	//2rd to calculate Hs(aR) + b
	//Hs(aR)
	Hs_aR := derivation2scalar(aR, outputIndex)

	//TODO:代码疑问
	//var onetimePriKey PrivKeyEd25519
	//onetimePriKeyAddr := (*[32]byte)(unsafe.Pointer(&onetimePriKey.Bytes()[0]))
	//edwards25519.ScAdd(onetimePriKeyAddr, Hs_aR, spendSecAddr)

	onetimePriKeydata := new([64]byte)
	onetimePriKeyAddr := (*[32]byte)(unsafe.Pointer(onetimePriKeydata))
	edwards25519.ScAdd(onetimePriKeyAddr, Hs_aR, spendSecAddr)

	prikey := PrivKeyPrivacy(*onetimePriKeydata)
	prikey.PubKey()
	return prikey, nil
}

//func GenerateKeyImage(onetimeSk PrivKey, onetimePk PubKey) *KeyImage {
//	keyImage := &KeyImage{}
//
//	return keyImage
//
//}
//其中的realUtxoIndex，是真实的utxo输出在混淆组中的位置索引
//func GenerateRingSignature(data []byte, utxos []*types.UTXO, realUtxoIndex int, sk []byte, keyImage []byte) Signature {
//
//	return nil
//}

//func CheckRingSignature(data, ringSignature []byte, groupUTXOPubKeys [][]byte, keyImage []byte) bool {
//	checkRes := false
//
//	return checkRes
//}

func GenerateKeyPair(privKeyPrivacyPtr *PrivKeyPrivacy, pubKeyPrivacyPtr *PubKeyPrivacy) {
	copy(privKeyPrivacyPtr[:PrivateKeyLen], CRandBytes(PrivateKeyLen))

	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(privKeyPrivacyPtr))
	addr64 := (*[PrivateKeyLen]byte)(unsafe.Pointer(privKeyPrivacyPtr))
	edwards25519.ScReduce(addr32, addr64)

	//to generate the publickey
	var A edwards25519.ExtendedGroupElement
	pubKeyAddr32 := (*[KeyLen32]byte)(unsafe.Pointer(pubKeyPrivacyPtr))
	edwards25519.GeScalarMultBase(&A, addr32)
	A.ToBytes(pubKeyAddr32)
	copy(addr64[KeyLen32:], pubKeyAddr32[:])
}

func generateKeyPairWithPrivKey(privByte *[KeyLen32]byte, privKeyPrivacyPtr *PrivKeyPrivacy, pubKeyPrivacyPtr *PubKeyPrivacy) error {
	if nil == privByte {
		return ErrNullRandInput
	}

	_, err := io.ReadFull(bytes.NewReader(privByte[:]), privKeyPrivacyPtr[:32])
	if err != nil {
		return err
	}

	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(privKeyPrivacyPtr))
	addr64 := (*[PrivateKeyLen]byte)(unsafe.Pointer(privKeyPrivacyPtr))
	edwards25519.ScReduce(addr32, addr64)

	//to generate the publickey
	var A edwards25519.ExtendedGroupElement
	pubKeyAddr32 := (*[KeyLen32]byte)(unsafe.Pointer(pubKeyPrivacyPtr))
	edwards25519.GeScalarMultBase(&A, addr32)
	A.ToBytes(pubKeyAddr32)
	copy(addr64[KeyLen32:], pubKeyAddr32[:])

	return nil
}

func mul8(r *edwards25519.CompletedGroupElement, t *edwards25519.ProjectiveGroupElement) {
	var u edwards25519.ProjectiveGroupElement
	t.Double(r)
	r.ToProjective(&u)
	u.Double(r)
	r.ToProjective(&u)
	u.Double(r)
}

func derivation2scalar(derivation_rA *[32]byte, outputIndex int64) (ellipticCurveScalar *[32]byte) {
	len := 32 + (unsafe.Sizeof(outputIndex)*8+6)/7
	//buf := new([len]byte)
	buf := make([]byte, len)
	copy(buf[:32], derivation_rA[:])
	index := 32
	for outputIndex >= 0x80 {
		buf[index] = byte((outputIndex & 0x7f) | 0x80)
		outputIndex >>= 7
		index++
	}
	buf[index] = byte(outputIndex & 0xff)

	//hash_to_scalar
	hash := sha3.Sum256(buf[:])
	digest := new([64]byte)
	copy(digest[:], hash[:])
	edwards25519.ScReduce(&hash, digest)

	return &hash
}

func hash2scalar(buf []byte, out *[32]byte) {
	hash := sha3.KeccakSum256(buf[:])
	digest := new([64]byte)
	copy(digest[:], hash[:])
	edwards25519.ScReduce(out, digest)
}

func (image *KeyImage) String() string {
	var s string
	for _, d := range image {
		s += fmt.Sprintf("%02x", d)
	}
	return s
}
