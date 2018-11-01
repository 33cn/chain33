package privacy

import (
	"bytes"
	"fmt"

	"gitlab.33.cn/chain33/chain33/common"
	. "gitlab.33.cn/chain33/chain33/common/crypto"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type PubKeyPrivacy [PublicKeyLen]byte

func (pubKey PubKeyPrivacy) Bytes() []byte {
	return pubKey[:]
}

func Bytes2PubKeyPrivacy(in []byte) PubKeyPrivacy {
	var temp PubKeyPrivacy
	copy(temp[:], in)
	return temp
}

func (pubKey PubKeyPrivacy) VerifyBytes(msg []byte, sig_ Signature) bool {
	var tx types.Transaction
	if err := types.Decode(msg, &tx); err != nil {
		return false
	}
	if privacytypes.PrivacyX != string(tx.Execer) {
		return false
	}
	var action privacytypes.PrivacyAction
	if err := types.Decode(tx.Payload, &action); err != nil {
		return false
	}
	var privacyInput *privacytypes.PrivacyInput
	if action.Ty == privacytypes.ActionPrivacy2Privacy && action.GetPrivacy2Privacy() != nil {
		privacyInput = action.GetPrivacy2Privacy().Input
	} else if action.Ty == privacytypes.ActionPrivacy2Public && action.GetPrivacy2Public() != nil {
		privacyInput = action.GetPrivacy2Public().Input
	} else {
		return false
	}
	var ringSign types.RingSignature
	if err := types.Decode(sig_.Bytes(), &ringSign); err != nil {
		return false
	}

	h := common.BytesToHash(msg)
	for i, ringSignItem := range ringSign.GetItems() {
		if !CheckRingSignature(h.Bytes(), ringSignItem, ringSignItem.Pubkey, privacyInput.Keyinput[i].KeyImage) {
			return false
		}
	}
	return true
}

//func (pubKey PubKeyPrivacy) VerifyBytes(msg []byte, sig_ Signature) bool {
//	// unwrap if needed
//	if wrap, ok := sig_.(SignatureS); ok {
//		sig_ = wrap.Signature
//	}
//	// make sure we use the same algorithm to sign
//	sig, ok := sig_.(SignatureOnetime)
//	if !ok {
//		return false
//	}
//	pubKeyBytes := [32]byte(pubKey)
//	sigBytes := [64]byte(sig)
//
//	var ege edwards25519.ExtendedGroupElement
//	if !ege.FromBytes(&pubKeyBytes) {
//		return false
//	}
//
//	sigAddr32a := (*[KeyLen32]byte)(unsafe.Pointer(&sigBytes[0]))
//	sigAddr32b := (*[KeyLen32]byte)(unsafe.Pointer(&sigBytes[32]))
//	if !edwards25519.ScCheck(sigAddr32a) || !edwards25519.ScCheck(sigAddr32b) {
//		return false
//	}
//
//	var sigcommdata sigcommArray
//	sigcommPtr := (*sigcomm)(unsafe.Pointer(&sigcommdata))
//	copy(sigcommPtr.pubkey[:], pubKey.Bytes())
//	hash := sha3.Sum256(msg)
//	copy(sigcommPtr.hash[:], hash[:])
//
//	var rge edwards25519.ProjectiveGroupElement
//	edwards25519.GeDoubleScalarMultVartime(&rge, sigAddr32a, &ege, sigAddr32b)
//	rge.ToBytes((*[KeyLen32]byte)(unsafe.Pointer(&sigcommPtr.comm[0])))
//
//	out32 := new([32]byte)
//	hash2scalar(sigcommdata[:], out32)
//	return subtle.ConstantTimeCompare(sigAddr32a[:], out32[:]) == 1
//}

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
