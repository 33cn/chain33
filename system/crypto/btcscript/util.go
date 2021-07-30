// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcscript

import (
	"errors"

	"github.com/golang/protobuf/proto"

	"github.com/33cn/chain33/common"
	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/btcsuite/btcutil"
)

const (
	// TyPay2PubKey Pay to Pubkey
	TyPay2PubKey = iota
	// TyPay2PubKeyHash Pay to Pubkey Hash
	TyPay2PubKeyHash
	// TyPay2ScriptHash Pay to Script Hash
	TyPay2ScriptHash
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

// SetSignOpt 设置签名参数
type SetSignOpt func(*btcSignOption)

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
	return btcec.PrivKeyFromBytes(btcec.S256(), priv)
}

// GetBtcLockScript 根据地址类型，生成锁定脚本
func GetBtcLockScript(scriptTy int32, pkScript []byte, params *chaincfg.Params) (btcutil.Address, []byte, error) {

	btcAddr, err := getBtcAddr(scriptTy, pkScript, params)
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
func GetBtcUnlockScript(btcTx *wire.MsgTx, lockScript, prevScript []byte,
	params *chaincfg.Params, kdb txscript.KeyDB, sdb txscript.ScriptDB) ([]byte, error) {

	sigScript, err := txscript.SignTxOutput(params, btcTx, 0,
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
	vm, err := txscript.NewEngine(sig.LockScript, tx, 0, txscript.StandardVerifyFlags, nil, nil, 0)
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
func NewBtcScriptSig(lockScript, unlockScript []byte, lockTime, utxoSeq int64) ([]byte, error) {
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

	tx := &wire.MsgTx{Version: ID, TxIn: []*wire.TxIn{{}}}
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

func mkGetKey(keyAddr ...*BtcAddr2Key) txscript.KeyDB {
	if len(keyAddr) <= 0 {
		return txscript.KeyClosure(func(addr btcutil.Address) (*btcec.PrivateKey,
			bool, error) {
			return nil, false, errors.New("mkGetKey:privKey not exist")
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
			return nil, false, errors.New("mkGetKey:privKey not exist")
		}
		return key, true, nil
	})
}

func mkGetScript(scriptArr ...*BtcAddr2Script) txscript.ScriptDB {

	if len(scriptArr) <= 0 {
		return txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
			return nil, errors.New("mkGetScript:Script not exist")
		})
	}
	scripts := make(map[string][]byte, len(scriptArr))
	for _, script := range scriptArr {
		scripts[script.Addr] = script.Script
	}
	return txscript.ScriptClosure(func(addr btcutil.Address) ([]byte, error) {
		script, ok := scripts[addr.EncodeAddress()]
		if !ok {
			return nil, errors.New("mkGetScript:Script not exist")
		}
		return script, nil
	})
}

// WithBtcLockScript 指定比特币锁定脚本以及对应的脚本编码地址
func WithBtcLockScript(lockScript []byte) SetSignOpt {
	return func(opt *btcSignOption) {
		opt.lockScript = lockScript
	}
}

// WithBtcPrivateKeys 指定私钥对应的编码地址，用于比特币签名kdb索引
func WithBtcPrivateKeys(keys ...*BtcAddr2Key) SetSignOpt {
	return func(opt *btcSignOption) {
		for _, key := range keys {
			opt.keys = append(opt.keys, key)
		}
	}
}

// WithBtcScripts 指定脚本对应的编码地址，用于比特币签名sdb索引
func WithBtcScripts(scripts ...*BtcAddr2Script) SetSignOpt {
	return func(opt *btcSignOption) {
		for _, script := range scripts {
			opt.scripts = append(opt.scripts, script)
		}
	}
}

// WithPreviousSigScript 分步签名脚本聚合, 例如多重签名不同人依次签名
func WithPreviousSigScript(prevScript []byte) SetSignOpt {
	return func(opt *btcSignOption) {
		opt.prevSigScript = prevScript
	}
}

// 设置签名相关参数
func applySignOption(option *btcSignOption, opts ...interface{}) {

	for _, opt := range opts {
		set, ok := opt.(SetSignOpt)
		if ok {
			set(option)
		}
	}
}

// 初始化默认参数，默认即采用Pay to PubKey
func initBtcSignOption(privateKey []byte) *btcSignOption {

	option := &btcSignOption{
		btcParams: Chain33BtcParams,
	}

	priv, pub := btcec.PrivKeyFromBytes(btcec.S256(), privateKey)
	addr, lockScript, err := GetBtcLockScript(TyPay2PubKey, pub.SerializeCompressed(), option.btcParams)
	if err != nil {
		panic("initBtcSignOption err: " + err.Error())
	}

	option.lockScript = lockScript
	ecAddr := addr.EncodeAddress()
	option.keys = append(option.keys, &BtcAddr2Key{Addr: ecAddr, Key: priv})
	return option
}
