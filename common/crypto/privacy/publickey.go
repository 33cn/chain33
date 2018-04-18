package privacy

import (
	"bytes"
	"fmt"
	"gitlab.33.cn/chain33/chain33/common"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	"golang.org/x/crypto/sha3"
	"unsafe"
)

type PubKeyPrivacy [PublicKeyLen]byte

func (pubKey PubKeyPrivacy) Bytes() []byte {
	return pubKey[:]
}

func (pubKey PubKeyPrivacy) VerifyBytes(msg []byte, sig_ Signature) bool {
	fmt.Printf("^^^^^^^^^^^Enter VerifyBytes process\n")
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

	fmt.Printf("^^^^^^^^^^^pubKeyBytes is:%X\n", pubKeyBytes[:])
	fmt.Printf("^^^^^^^^^^^sigBytes is:%X\n", sigBytes[:])

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
	fmt.Printf("^^^^^^^^^^^finish VerifyBytes process\n")
	fmt.Printf("^^^^^^^^^^^sigAddr32a:%s\n", common.Bytes2Hex(sigAddr32a[:]))
	fmt.Printf("^^^^^^^^^^^out32:     %s\n", common.Bytes2Hex(out32[:]))

	s := new([32]byte)
	edwards25519.ScSub(s, sigAddr32a, out32)

	fmt.Printf("^^^^^^^^^^^ScSub:     %x\n", s[:])

	compareRes := (((int)(s[0]|s[1]|s[2]|s[3]|s[4]|s[5]|s[6]|s[7]|s[8]|
		s[9]|s[10]|s[11]|s[12]|s[13]|s[14]|s[15]|s[16]|s[17]|
		s[18]|s[19]|s[20]|s[21]|s[22]|s[23]|s[24]|s[25]|s[26]|
		s[27]|s[28]|s[29]|s[30]|s[31]) - 1) >> 8) + 1

	return compareRes == 0
	//return subtle.ConstantTimeCompare(sigAddr32a[:], out32[:]) == 1
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
