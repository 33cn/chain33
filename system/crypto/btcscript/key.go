// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcscript

import (
	"bytes"
	"fmt"

	"github.com/33cn/chain33/common/crypto"
	secp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/gogo/protobuf/proto"
)

const (
	TyPay2PubKey = iota
	TyPay2Script
)

var btcParams = &chaincfg.Params{
	Name: "chain33-btc-script",
	//PubKeyHashAddrID:
}

type signOption struct {
	ty         int32
	btcScript  []byte
	prevScript []byte
}

//privKeyBtcScript PrivKey
type privKeyBtcScript struct {
	key     [32]byte
	signOpt signOption
}

//Bytes 字节格式
func (priv privKeyBtcScript) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, priv.key[:])
	return s
}

//Sign 签名
func (priv privKeyBtcScript) Sign(msg []byte, opts ...interface{}) crypto.Signature {

	var err error
	key, pk := secp256k1.PrivKeyFromBytes(secp256k1.S256(), priv.key[:])

	btcAddr, lockScript, err := GetBtcLockScript(priv.signOpt.ty, pk.SerializeCompressed(), priv.signOpt.btcScript)
	if err != nil {
		panic("GetBtcLockScript err:" + err.Error())
	}

	unlockScript, err := GetBtcUnlockScript(msg, lockScript, priv.signOpt.prevScript, txscript.SigHashAll, btcParams,
		mkGetKey(map[string]addressToKey{btcAddr.EncodeAddress(): {key, true}}),
		mkGetScript(map[string][]byte{btcAddr.EncodeAddress(): priv.signOpt.btcScript}))

	if err != nil {
		panic("sign btc script err:" + err.Error())
	}
	sig, err := proto.Marshal(&Signature{
		ScriptTy:     priv.signOpt.ty,
		UnlockScript: unlockScript,
	})

	if err != nil {
		panic("serial signature err:" + err.Error())
	}

	return sigBtcScript(sig)
}

//PubKey 私钥生成公钥
func (priv privKeyBtcScript) PubKey(opts ...interface{}) crypto.PubKey {
	_, pub := secp256k1.PrivKeyFromBytes(secp256k1.S256(), priv.key[:])
	var pubSecp256k1 pubKeyBtcScript
	copy(pubSecp256k1[:], pub.SerializeCompressed())
	return pubSecp256k1
}

//Equals 私钥是否相等
func (priv privKeyBtcScript) Equals(other crypto.PrivKey) bool {
	if otherPriv, ok := other.(privKeyBtcScript); ok {
		return bytes.Equal(priv.key[:], otherPriv.key[:])
	}
	return false

}

func (priv privKeyBtcScript) String() string {
	return "privKeyBtcScript{*****}"
}

//pubKeyBtcScript Compressed pubkey (just the x-cord),
// prefixed with 0x02 or 0x03, depending on the y-cord.
type pubKeyBtcScript [33]byte

//Bytes 字节格式
func (pubKey pubKeyBtcScript) Bytes() []byte {
	s := make([]byte, 33)
	copy(s, pubKey[:])
	return s
}

//VerifyBytes 公钥不做验证，统一调用driver validate接口
func (pubKey pubKeyBtcScript) VerifyBytes(msg []byte, sig crypto.Signature) bool {
	return false
}

func (pubKey pubKeyBtcScript) String() string {
	return fmt.Sprintf("pubKeyBtcScript{%X}", pubKey[:])
}

//KeyString Must return the full bytes in hex.
// Used for map keying, etc.
func (pubKey pubKeyBtcScript) KeyString() string {
	return fmt.Sprintf("%X", pubKey[:])
}

//Equals 公钥相等
func (pubKey pubKeyBtcScript) Equals(other crypto.PubKey) bool {
	if otherSecp, ok := other.(pubKeyBtcScript); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	}
	return false

}

//sigBtcScript sigBtcScript
type sigBtcScript []byte

//Bytes 字节格式
func (sig sigBtcScript) Bytes() []byte {
	s := make([]byte, len(sig))
	copy(s, sig[:])
	return s
}

//IsZero 是否是0
func (sig sigBtcScript) IsZero() bool { return len(sig) == 0 }

func (sig sigBtcScript) String() string {
	fingerprint := make([]byte, len(sig[:]))
	copy(fingerprint, sig[:])
	return fmt.Sprintf("/%X.../", fingerprint)

}

//Equals 相等
func (sig sigBtcScript) Equals(other crypto.Signature) bool {
	if otherEd, ok := other.(sigBtcScript); ok {
		return bytes.Equal(sig[:], otherEd[:])
	}
	return false

}
