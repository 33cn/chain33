// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package btcscript btc script driver
package btcscript

import (
	"errors"

	"github.com/33cn/chain33/common/crypto"
	secp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/txscript"
	"github.com/gogo/protobuf/proto"
)

//const
const (
	Name = "btcscript"
	ID   = 11
)

func init() {
	// 默认启用高度-1， 不开启
	crypto.Register(Name, &Driver{}, crypto.WithRegOptionTypeID(ID))
}

//Driver 驱动
type Driver struct{}

//GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	privKeyBytes := [32]byte{}
	copy(privKeyBytes[:], crypto.CRandBytes(32))
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return privKeyBtcScript{key: privKeyBytes}, nil
}

//PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	if len(b) != 32 {
		return nil, errors.New("invalid priv key byte")
	}
	privKeyBytes := new([32]byte)
	copy(privKeyBytes[:], b[:32])
	priv, _ := secp256k1.PrivKeyFromBytes(secp256k1.S256(), privKeyBytes[:])
	copy(privKeyBytes[:], priv.Serialize())
	return privKeyBtcScript{key: *privKeyBytes}, nil
}

//PubKeyFromBytes 字节转为公钥
func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	if len(b) != 33 {
		return nil, errors.New("invalid pub key byte")
	}
	pubKeyBytes := new([33]byte)
	copy(pubKeyBytes[:], b[:])
	return pubKeyBtcScript(*pubKeyBytes), nil
}

//SignatureFromBytes 字节转为签名
func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	return sigBtcScript(b), nil
}

// Validate validate msg and signature
func (d Driver) Validate(msg, pk, sig []byte) error {
	ssig := &Signature{}
	err := proto.Unmarshal(sig, ssig)
	if err != nil {
		return err
	}

	_, lockScript, err := GetBtcLockScript(ssig.ScriptTy, pk, pk)
	if err != nil {
		return err
	}

	return CheckBtcScript(msg, lockScript, ssig.UnlockScript, txscript.StandardVerifyFlags)
}
