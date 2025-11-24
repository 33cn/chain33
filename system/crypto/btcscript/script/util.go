// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package script

import (
	"errors"

	"github.com/golang/protobuf/proto"

	"github.com/33cn/chain33/common"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

const (
	// TyPay2PubKey Pay to pub key
	TyPay2PubKey = iota
	// TyPay2PubKeyHash Pay to pub key Hash
	TyPay2PubKeyHash
	// TyPay2ScriptHash Pay to Script Hash
	TyPay2ScriptHash

	// 比特币交易版本
	btcTxVersion = 10
)

// Chain33BtcParams 比特币相关区块链参数
var Chain33BtcParams = &chaincfg.Params{
	Name: "chain33-btc-Script",

	// Address encoding magics, bitcoin main net params
	PubKeyHashAddrID:        0x00, // starts with 1
	ScriptHashAddrID:        0x05, // starts with 3
	PrivateKeyID:            0x80, // starts with 5 (uncompressed) or K (compressed)
	WitnessPubKeyHashAddrID: 0x06, // starts with p2
	WitnessScriptHashAddrID: 0x0A, // starts with 7Xh
}

// BtcAddr2Key 比特币编码地址及对应的私钥，用于签名KeyDB
type BtcAddr2Key struct {
	Addr string
	Key  *btcec.PrivateKey
}

// BtcAddr2Script 比特币编码地址及对应的脚本，用于签名ScriptDB
type BtcAddr2Script struct {
	Addr   string
	Script []byte
}

// NewBtcKeyFromBytes 获取比特币公私钥
func NewBtcKeyFromBytes(priv []byte) (*btcec.PrivateKey, *btcec.PublicKey) {
	return btcec.PrivKeyFromBytes(priv)
}

// GetBtcLockScript 根据地址类型，生成锁定脚本
func GetBtcLockScript(scriptTy int32, pkScript []byte) (btcutil.Address, []byte, error) {

	btcAddr, err := getBtcAddr(scriptTy, pkScript, Chain33BtcParams)
	if err != nil {
		return nil, nil, errors.New("get btc Addr err:" + err.Error())
	}

	lockScript, err := txscript.PayToAddrScript(btcAddr)
	if err != nil {
		return nil, nil, errors.New("get pay to Addr Script err:" + err.Error())
	}
	return btcAddr, lockScript, nil
}

func getBtcAddr(scriptTy int32, pkScript []byte, params *chaincfg.Params) (btcutil.Address, error) {
	if scriptTy == TyPay2PubKey {
		return btcutil.NewAddressPubKey(pkScript, params)
	} else if scriptTy == TyPay2PubKeyHash {
		return btcutil.NewAddressPubKeyHash(btcutil.Hash160(pkScript), params)
	} else if scriptTy == TyPay2ScriptHash {
		return btcutil.NewAddressScriptHash(pkScript, params)
	}
	return nil, errors.New("InvalidScriptType")
}

// GetBtcUnlockScript 生成比特币解锁脚本
func GetBtcUnlockScript(signMsg, lockScript, prevScript []byte,
	kdb txscript.KeyDB, sdb txscript.ScriptDB) ([]byte, error) {

	btcTx := getBindBtcTx(signMsg)
	sigScript, err := txscript.SignTxOutput(Chain33BtcParams, btcTx, 0,
		lockScript, txscript.SigHashAll, kdb, sdb, prevScript)
	if err != nil {
		return nil, errors.New("sign btc tx output err:" + err.Error())
	}
	return sigScript, nil
}

// CheckBtcScript check btc Script signature
func CheckBtcScript(msg []byte, sig *Signature) error {

	tx := getBindBtcTx(msg)
	setBtcTx(tx, sig.LockTime, sig.UtxoSequence, sig.UnlockScript)
	vm, err := txscript.NewEngine(sig.LockScript, tx, 0, txscript.StandardVerifyFlags, nil, nil, 0, nil)
	if err != nil {
		return errors.New("new Script engine err:" + err.Error())
	}

	err = vm.Execute()
	if err != nil {
		return errors.New("execute engine err:" + err.Error())
	}
	return nil
}

// NewBtcScriptSig new btc script signature
func NewBtcScriptSig(lockScript, unlockScript []byte) ([]byte, error) {
	return newBtcScriptSig(lockScript, unlockScript, 0, 0)
}

// NewBtcScriptSigWithDelay new btc script signature with lockTime or sequence
func NewBtcScriptSigWithDelay(lockScript, unlockScript []byte, lockTime, utxoSeq int64) ([]byte, error) {
	return newBtcScriptSig(lockScript, unlockScript, lockTime, utxoSeq)
}

// NewBtcScriptSig new btc script signature
func newBtcScriptSig(lockScript, unlockScript []byte, lockTime, utxoSeq int64) ([]byte, error) {
	sig := &Signature{
		LockScript:   lockScript,
		UnlockScript: unlockScript,
		LockTime:     lockTime,
		UtxoSequence: utxoSeq,
	}

	return proto.Marshal(sig)
}

// Script2PubKey transform script to fixed length public key
func Script2PubKey(lockScript []byte) []byte {
	return common.Sha256(lockScript)
}

// 比特币脚本签名依赖原生交易结构，这里构造一个带一个输入的伪交易
// HACK: 通过构造临时比特币交易，将第一个输入的chainHash设为签名数据的哈希，完成绑定关系
func getBindBtcTx(msg []byte) *wire.MsgTx {

	tx := &wire.MsgTx{Version: btcTxVersion, TxIn: []*wire.TxIn{{}}}
	_ = tx.TxIn[0].PreviousOutPoint.Hash.SetBytes(common.Sha256(msg)[:chainhash.HashSize])
	return tx
}

// set btc tx
func setBtcTx(tx *wire.MsgTx, lockTime, utxoSequence int64, sigScript []byte) {

	if lockTime > 0 {
		tx.LockTime = uint32(lockTime)
	}
	if utxoSequence > 0 {
		tx.TxIn[0].Sequence = uint32(utxoSequence)
	}
	if len(sigScript) > 0 {
		tx.TxIn[0].SignatureScript = sigScript
	}
}

// MakeKeyDB make btc script key db
func MakeKeyDB(keyAddr ...*BtcAddr2Key) txscript.KeyDB {
	if len(keyAddr) <= 0 {
		return txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey,
			bool, error) {
			return nil, false, ErrBtcKeyNotExist
		})
	}
	keys := make(map[string]*btcec.PrivateKey, len(keyAddr))
	for _, key := range keyAddr {
		keys[key.Addr] = key.Key
	}
	return txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey,
		bool, error) {
		key, ok := keys[addr.EncodeAddress()]
		if !ok {
			return nil, false, ErrBtcKeyNotExist
		}
		return key, true, nil
	})
}

// MakeScriptDB make btc script db
func MakeScriptDB(scriptArr ...*BtcAddr2Script) txscript.ScriptDB {

	if len(scriptArr) <= 0 {
		return txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
			return nil, ErrBtcScriptNotExist
		})
	}
	scripts := make(map[string][]byte, len(scriptArr))
	for _, script := range scriptArr {
		scripts[script.Addr] = script.Script
	}
	return txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
		script, ok := scripts[addr.EncodeAddress()]
		if !ok {
			return nil, ErrBtcScriptNotExist
		}
		return script, nil
	})
}
