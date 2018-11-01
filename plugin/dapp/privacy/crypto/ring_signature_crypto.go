/*
基于框架中Crypto接口，实现签名、验证的处理
*/
package privacy

import (
	"bytes"
	"fmt"
	"unsafe"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/ed25519/edwards25519"
	privacytypes "gitlab.33.cn/chain33/chain33/plugin/dapp/privacy/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	crypto.Register(privacytypes.SignNameRing, &RingSignED25519{})
}

// RingSignature 环签名中对于crypto.Signature接口实现
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
	if _, ok := other.(*RingSignature); ok {
		this := types.Encode(&r.sign)
		return bytes.Equal(this, other.Bytes())
	}
	return false
}

// RingSignPrivateKey 环签名中对于crypto.PrivKey接口实现
type RingSignPrivateKey struct {
	key [PrivateKeyLen]byte
}

func (privkey *RingSignPrivateKey) Bytes() []byte {
	return privkey.key[:]
}

func (privkey *RingSignPrivateKey) Sign(msg []byte) crypto.Signature {
	emptySign := &RingSignature{}
	if len(msg) <= 0 {
		return emptySign
	}
	tx := new(types.Transaction)
	if err := types.Decode(msg, tx); err != nil || !bytes.Equal([]byte(privacytypes.PrivacyX), tx.Execer) {
		// 目前只有隐私交易用到了环签名
		return emptySign
	}
	action := new(privacytypes.PrivacyAction)
	if err := types.Decode(tx.Payload, action); err != nil {
		return emptySign
	}
	if action.Ty != privacytypes.ActionPrivacy2Privacy && action.Ty != privacytypes.ActionPrivacy2Public {
		// 隐私交易中，隐私到隐私，隐私到公开才用到环签名
		return emptySign
	}
	//
	privacyInput := action.GetInput()
	retSign := new(types.RingSignature)
	if err := types.Decode(tx.Signature.Signature, retSign); err != nil {
		// 目前只有隐私交易用到了环签名
		return emptySign
	}
	//data := types.Encode(tx)
	//h := common.BytesToHash(data)
	for i, keyinput := range privacyInput.Keyinput {
		utxos := new(privacytypes.UTXOBasics)
		for _, item := range retSign.Items {
			utxo := new(privacytypes.UTXOBasic)
			utxo.OnetimePubkey = item.Pubkey[i]
			utxo.UtxoGlobalIndex = keyinput.UtxoGlobalIndex[i]
			utxos.Utxos = append(utxos.Utxos, utxo)
		}
		//
		//item, err := GenerateRingSignature(
		//	h.Bytes(),
		//	utxos.Utxos,
		//	realkeyInputSlice[i].Onetimeprivkey,
		//	int(realkeyInputSlice[i].Realinputkey),
		//	keyinput.KeyImage)
		//if err != nil {
		//	return emptySign
		//}
		//retSign.sign.Items = append(retSign.sign.Items, item)
	}
	return emptySign
}

func (privkey *RingSignPrivateKey) PubKey() crypto.PubKey {
	publicKey := new(RingSignPublicKey)
	addr32 := (*[KeyLen32]byte)(unsafe.Pointer(&privkey.key))
	addr64 := (*[PrivateKeyLen]byte)(unsafe.Pointer(&privkey.key))

	A := new(edwards25519.ExtendedGroupElement)
	edwards25519.GeScalarMultBase(A, addr32)
	A.ToBytes(&publicKey.key)
	// 这里可能有问题
	copy(addr64[KeyLen32:], publicKey.key[:])
	return publicKey
}

func (privkey *RingSignPrivateKey) Equals(other crypto.PrivKey) bool {
	if otherPrivKey, ok := other.(*RingSignPrivateKey); ok {
		return bytes.Equal(privkey.key[:], otherPrivKey.key[:])
	}
	return false
}

// RingSignPublicKey 环签名中对于crypto.PubKey接口实现
type RingSignPublicKey struct {
	key [PublicKeyLen]byte
}

func (pubkey *RingSignPublicKey) Bytes() []byte {
	return pubkey.key[:]
}

func (pubkey *RingSignPublicKey) VerifyBytes(msg []byte, sign crypto.Signature) bool {
	if len(msg) <= 0 {
		return false
	}
	ringSign := new(types.RingSignature)
	if err := types.Decode(sign.Bytes(), ringSign); err != nil {
		return false
	}
	tx := new(types.Transaction)
	if err := types.Decode(msg, tx); err != nil || !bytes.Equal([]byte(privacytypes.PrivacyX), tx.Execer) {
		// 目前只有隐私交易用到了环签名
		return false
	}
	action := new(privacytypes.PrivacyAction)
	if err := types.Decode(tx.Payload, action); err != nil {
		return false
	}
	if action.Ty != privacytypes.ActionPrivacy2Privacy && action.Ty != privacytypes.ActionPrivacy2Public {
		// 隐私交易中，隐私到隐私，隐私到公开才用到环签名
		return false
	}
	input := action.GetInput()
	if len(input.Keyinput) != len(ringSign.Items) {
		return false
	}
	h := common.BytesToHash(msg)
	for i, ringSignItem := range ringSign.GetItems() {
		if !CheckRingSignature(h.Bytes(), ringSignItem, ringSignItem.Pubkey, input.Keyinput[i].KeyImage) {
			return false
		}
	}
	return true
}

func (pubkey *RingSignPublicKey) KeyString() string {
	return fmt.Sprintf("%X", pubkey.key[:])
}

func (pubkey *RingSignPublicKey) Equals(other crypto.PubKey) bool {
	if otherPubKey, ok := other.(*RingSignPublicKey); ok {
		return bytes.Equal(pubkey.key[:], otherPubKey.key[:])
	}
	return false
}

// RingSignED25519 对应crypto.Crypto的接口实现
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
	if len(b) <= 0 {
		return nil, types.ErrInvalidParam
	}
	if len(b) != KeyLen32 {
		return nil, types.ErrPrivateKeyLen
	}
	privateKey := new(RingSignPrivateKey)
	copy(privateKey.key[:], b)
	return privateKey, nil
}

func (r *RingSignED25519) PubKeyFromBytes(b []byte) (crypto.PubKey, error) {
	if len(b) <= 0 {
		return nil, types.ErrInvalidParam
	}
	if len(b) != PublicKeyLen {
		return nil, types.ErrPubKeyLen
	}
	publicKey := new(RingSignPublicKey)
	copy(publicKey.key[:], b)
	return publicKey, nil
}

func (r *RingSignED25519) SignatureFromBytes(b []byte) (crypto.Signature, error) {
	if len(b) <= 0 {
		return nil, types.ErrInvalidParam
	}
	sign := new(RingSignature)
	if err := types.Decode(b, &sign.sign); err != nil {
		return nil, err
	}
	return sign, nil
}
