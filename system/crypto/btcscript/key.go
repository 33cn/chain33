// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcscript

import (
	"bytes"
	"fmt"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/golang/protobuf/proto"
)

type btcSignOption struct {
	prevSigScript []byte
	lockScript    []byte
	keys          map[string]*BtcAddr2Key
	scripts       map[string]*BtcAddr2Script
	btcParams     *chaincfg.Params
}

//privKeyBtcScript PrivKey
type privKeyBtcScript struct {
	key [32]byte
}

//Bytes 字节格式
func (priv privKeyBtcScript) Bytes() []byte {
	s := make([]byte, 32)
	copy(s, priv.key[:])
	return s
}

//Sign 签名
func (priv privKeyBtcScript) Sign(msg []byte, opts ...interface{}) crypto.Signature {

	option := initBtcSignOption(priv.key[:])
	applySignOption(option, opts...)

	unlockScript, err := GetBtcUnlockScript(msg, option.lockScript,
		option.prevSigScript, txscript.SigHashAll, Chain33BtcParams,
		mkGetKey(option.keys), mkGetScript(option.scripts))

	if err != nil {
		panic("GetBtcUnlockScript err:" + err.Error())
	}
	sig, err := proto.Marshal(&Signature{
		LockScript:   option.lockScript,
		UnlockScript: unlockScript,
	})

	if err != nil {
		panic("marshal signature err:" + err.Error())
	}

	return sigBtcScript(sig)
}

//PubKey 私钥生成公钥, 公钥由锁定脚本生成
func (priv privKeyBtcScript) PubKey(opts ...interface{}) crypto.PubKey {

	option := initBtcSignOption(priv.key[:])
	applySignOption(option, opts...)
	return pubKeyBtcScript(common.Sha256(option.lockScript))
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
type pubKeyBtcScript []byte

//Bytes 字节格式
func (pubKey pubKeyBtcScript) Bytes() []byte {
	s := make([]byte, len(pubKey))
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
