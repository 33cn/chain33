// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// script_test btc script test example
package btcscript

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/common"
	nty "github.com/33cn/chain33/system/dapp/none/types"

	"github.com/33cn/chain33/client/mocks"
	cryptocli "github.com/33cn/chain33/common/crypto/client"
	"github.com/stretchr/testify/mock"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"

	"github.com/33cn/chain33/system/crypto/btcscript/script"
	_ "github.com/33cn/chain33/system/dapp/init"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

// 转账到公钥
func Test_ExamplePay2PubKey(t *testing.T) {

	d := Driver{}
	priv, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)

	btcPriv, btcPub := script.NewBtcKeyFromBytes(priv.Bytes())

	btcAddr, lockScript, err := script.GetBtcLockScript(script.TyPay2PubKey, btcPub.SerializeCompressed())
	require.Nil(t, err)
	unlockScript, err := script.GetBtcUnlockScript(signMsg, lockScript, nil,
		script.MakeKeyDB(&script.BtcAddr2Key{
			Addr: btcAddr.EncodeAddress(),
			Key:  btcPriv,
		}), nil)
	require.Nil(t, err)
	sig, err := script.NewBtcScriptSig(lockScript, unlockScript)
	require.Nil(t, err)
	tx.Signature = &types.Signature{
		Ty:        ID,
		Signature: sig,
		Pubkey:    script.Script2PubKey(lockScript),
	}

	require.True(t, tx.CheckSign(0))
	//invalid pub key
	tx.Signature.Pubkey = priv.PubKey().Bytes()
	require.False(t, tx.CheckSign(0))
}

// 转账到地址
func Test_ExamplePay2PubKeyHash(t *testing.T) {

	d := Driver{}
	priv, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)

	btcPriv, btcPub := script.NewBtcKeyFromBytes(priv.Bytes())

	btcAddr, lockScript, err := script.GetBtcLockScript(script.TyPay2PubKeyHash, btcPub.SerializeCompressed())
	require.Nil(t, err)
	unlockScript, err := script.GetBtcUnlockScript(signMsg, lockScript, nil,
		script.MakeKeyDB(&script.BtcAddr2Key{
			Addr: btcAddr.EncodeAddress(),
			Key:  btcPriv,
		}), nil)
	require.Nil(t, err)
	sig, err := script.NewBtcScriptSig(lockScript, unlockScript)
	require.Nil(t, err)
	tx.Signature = &types.Signature{
		Ty:        ID,
		Signature: sig,
		Pubkey:    script.Script2PubKey(lockScript),
	}

	require.True(t, tx.CheckSign(0))
	tx.Signature.Pubkey = priv.PubKey().Bytes()
	require.False(t, tx.CheckSign(0))
}

// 转账到脚本
func Test_ExamplePay2ScriptHash(t *testing.T) {

	d := Driver{}
	priv, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)

	btcPriv, btcPub := script.NewBtcKeyFromBytes(priv.Bytes())

	btcAddr, pkScript, err := script.GetBtcLockScript(script.TyPay2PubKeyHash, btcPub.SerializeCompressed())
	require.Nil(t, err)

	scriptAddr, lockScript, err := script.GetBtcLockScript(script.TyPay2ScriptHash, pkScript)
	require.Nil(t, err)

	unlockScript, err := script.GetBtcUnlockScript(signMsg, lockScript, nil,
		script.MakeKeyDB(&script.BtcAddr2Key{
			Addr: btcAddr.EncodeAddress(),
			Key:  btcPriv,
		}), script.MakeScriptDB(&script.BtcAddr2Script{
			Addr:   scriptAddr.EncodeAddress(),
			Script: pkScript,
		}))
	require.Nil(t, err)

	sig, err := script.NewBtcScriptSig(lockScript, unlockScript)
	require.Nil(t, err)
	tx.Signature = &types.Signature{
		Ty:        ID,
		Signature: sig,
		Pubkey:    script.Script2PubKey(lockScript),
	}

	require.True(t, tx.CheckSign(0))
	// invalid pub key
	tx.Signature.Pubkey = priv.PubKey().Bytes()
	require.False(t, tx.CheckSign(0))
}

// 多重签名
func Test_ExampleMultiSig(t *testing.T) {

	d := Driver{}
	priv1, err := d.GenKey()
	require.Nil(t, err)
	priv2, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)
	key1, pk1 := script.NewBtcKeyFromBytes(priv1.Bytes())
	key2, pk2 := script.NewBtcKeyFromBytes(priv2.Bytes())
	addr1, err := btcutil.NewAddressPubKey(pk1.SerializeCompressed(), script.Chain33BtcParams)
	require.Nil(t, err)

	addr2, err := btcutil.NewAddressPubKey(pk2.SerializeCompressed(), script.Chain33BtcParams)
	require.Nil(t, err)

	pkScript, err := txscript.MultiSigScript([]*btcutil.AddressPubKey{addr1, addr2}, 2)
	require.Nil(t, err)

	scriptAddr, lockScript, err := script.GetBtcLockScript(script.TyPay2ScriptHash, pkScript)
	require.Equal(t, nil, err)

	// 只进行其中一个账户签名
	unlockScript, err := script.GetBtcUnlockScript(signMsg, lockScript, nil,
		script.MakeKeyDB(&script.BtcAddr2Key{
			Addr: addr1.EncodeAddress(),
			Key:  key1,
		}), script.MakeScriptDB(&script.BtcAddr2Script{
			Addr:   scriptAddr.EncodeAddress(),
			Script: pkScript,
		}))
	require.Nil(t, err)

	sig, err := script.NewBtcScriptSig(lockScript, unlockScript)
	require.Nil(t, err)
	tx.Signature = &types.Signature{
		Ty:        ID,
		Signature: sig,
		Pubkey:    script.Script2PubKey(lockScript),
	}
	// 只进行了其中一个多签账户签名，不满足2:2要求，验证失败
	require.False(t, tx.CheckSign(0))

	// 在地址1基础上进行地址2的签名
	unlockScript, err = script.GetBtcUnlockScript(signMsg, lockScript, unlockScript,
		script.MakeKeyDB(&script.BtcAddr2Key{
			Addr: addr2.EncodeAddress(),
			Key:  key2,
		}), script.MakeScriptDB(&script.BtcAddr2Script{
			Addr:   scriptAddr.EncodeAddress(),
			Script: pkScript,
		}))
	require.Nil(t, err)
	sig, err = script.NewBtcScriptSig(lockScript, unlockScript)
	require.Nil(t, err)
	tx.Signature.Signature = sig
	require.True(t, tx.CheckSign(0))

	// 单步多签，多个私钥同时签名
	unlockScript, err = script.GetBtcUnlockScript(signMsg, lockScript, nil,
		script.MakeKeyDB(&script.BtcAddr2Key{
			Addr: addr1.EncodeAddress(),
			Key:  key1,
		}, &script.BtcAddr2Key{
			Addr: addr2.EncodeAddress(),
			Key:  key2,
		}),
		script.MakeScriptDB(&script.BtcAddr2Script{
			Addr:   scriptAddr.EncodeAddress(),
			Script: pkScript,
		}))
	require.Nil(t, err)
	sig, err = script.NewBtcScriptSig(lockScript, unlockScript)
	require.Nil(t, err)
	tx.Signature.Signature = sig
	require.True(t, tx.CheckSign(0))

	// invalid pub key
	tx.Signature.Pubkey = priv1.PubKey().Bytes()
	require.False(t, tx.CheckSign(1))
}

func TestDriver_Validate(t *testing.T) {

	d := Driver{}
	err := d.Validate(nil, nil, []byte("test"))
	require.Equal(t, errDecodeBtcSignature, err)
	sig := &script.Signature{}
	sig.LockScript = []byte("testlockscript")
	err = d.Validate(nil, []byte("testpub"), types.Encode(sig))
	require.Equal(t, errInvalidLockScript, err)
	sig.LockTime = 11
	cryptocli.SetCurrentBlock(0, 10)
	pubKey := script.Script2PubKey(sig.LockScript)
	err = d.Validate(nil, pubKey, types.Encode(sig))
	require.Equal(t, errInvalidLockTime, err)
	sig.LockTime = 10
	sig.UtxoSequence = 10

	err = d.Validate([]byte("testmsg"), pubKey, types.Encode(sig))
	require.Equal(t, errDecodeValidateTx, err)

	api := &mocks.QueueProtocolAPI{}
	cryptocli.SetQueueAPI(api)
	txMsg := types.Encode(&types.Transaction{Execer: []byte("none")})
	api.On("Query", mock.Anything, mock.Anything,
		mock.Anything).Return(nil, errQueryDelayBeginTime).Once()
	err = d.Validate(txMsg, pubKey, types.Encode(sig))
	require.Equal(t, errQueryDelayBeginTime, err)

	api.On("Query", mock.Anything, mock.Anything,
		mock.Anything).Return(nil, nil).Once()
	err = d.Validate(txMsg, pubKey, types.Encode(sig))
	require.Equal(t, errQueryDelayBeginTime, err)

	api.On("Query", mock.Anything, mock.Anything,
		mock.Anything).Return(&nty.CommitDelayTxLog{DelayBeginTimestamp: 1}, nil)
	err = d.Validate(txMsg, pubKey, types.Encode(sig))
	require.Equal(t, errInvalidUtxoSequence, err)

	cryptocli.SetCurrentBlock(1, 11)
	err = d.Validate(txMsg, pubKey, types.Encode(sig))
	require.Equal(t, errInvalidBtcSignature, err)
}

var (
	addr1    = "1MUnpSyvPuB1tie8hCGgWzjwVtHw7sNXRV"
	privHex1 = "0xf8e994ba1d3577ebbde5ae0264df830327eb3a4187277d62e3e580afef9102b3"
	addr2    = "1Ka7EPFRqs3v9yreXG6qA4RQbNmbPJCZPj"
	privHex2 = "0xd165c84ed37c2a427fea487470ee671b7a0495d68d82607cafbc6348bf23bec5"
	//wrAddr   = "16zUnbHEFfEsHdsJj6FEHJ6f7dMowTBsBP"
)

// wallet recover transaction example
func Test_wallet_recover_transaction(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode.")
	}
	privBytes1, _ := common.FromHex(privHex1)
	privBytes2, _ := common.FromHex(privHex2)
	d := Driver{}
	priv1, err := d.PrivKeyFromBytes(privBytes1)
	require.Nil(t, err)
	priv2, err := d.PrivKeyFromBytes(privBytes2)
	require.Nil(t, err)

	delay := int64(10)
	wrScript, err := script.NewWalletRecoveryScript(priv1.PubKey().Bytes(),
		[][]byte{priv2.PubKey().Bytes()}, delay)
	require.Nil(t, err)
	//wrAddr := address.PubKeyToAddr(script.Script2PubKey(wrScript))
	//println(wrAddr)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx1 := util.CreateCoinsTx(cfg, priv1, addr1, 9*types.DefaultCoinPrecision)
	tx1.Signature = nil
	tx1.ChainID = 0
	tx1.Nonce = types.Now().UnixNano()
	sig, pk, err := script.GetWalletRecoverySignature(false, types.Encode(tx1),
		priv1.Bytes(), wrScript, delay)
	require.Nil(t, err)
	tx1.Signature = &types.Signature{
		Ty:        ID,
		Pubkey:    pk,
		Signature: sig,
	}

	tx1Hex := hex.EncodeToString(types.Encode(tx1))
	println("tx1: " + tx1Hex)

	tx2 := util.CreateCoinsTx(cfg, priv1, addr2, 10*types.DefaultCoinPrecision)
	tx2.Signature = nil
	tx2.ChainID = 0
	tx2.Nonce = types.Now().UnixNano()
	sig, pk, err = script.GetWalletRecoverySignature(true, types.Encode(tx2),
		priv2.Bytes(), wrScript, delay)
	require.Nil(t, err)
	tx2.Signature = &types.Signature{
		Ty:        ID,
		Pubkey:    pk,
		Signature: sig,
	}

	tx2Hex := hex.EncodeToString(types.Encode(tx2))
	println("tx2: " + tx2Hex)

	tx3 := util.CreateNoneTx(cfg, priv1)
	tx3.ChainID = 0
	action := &nty.NoneAction{Ty: nty.TyCommitDelayTxAction}
	action.Value = &nty.NoneAction_CommitDelayTx{CommitDelayTx: &nty.CommitDelayTx{
		DelayTx:             tx2Hex,
		RelativeDelayHeight: delay,
	}}
	tx3.Payload = types.Encode(action)
	tx3.Sign(types.SECP256K1, priv1)
	tx3Hex := hex.EncodeToString(types.Encode(tx3))
	println("tx3: " + tx3Hex)
}
