// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// script_test btc script test example
package btcscript

import (
	"testing"

	"github.com/33cn/chain33/client/mocks"
	cryptocli "github.com/33cn/chain33/common/crypto/client"
	"github.com/stretchr/testify/mock"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"

	"github.com/33cn/chain33/system/crypto/btcscript/script"
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
	sig.LockTime = 100
	cryptocli.SetCurrentBlock(10, types.Now().Unix())
	pubKey := script.Script2PubKey(sig.LockScript)
	err = d.Validate(nil, pubKey, types.Encode(sig))
	require.Equal(t, errInvalidLockTime, err)
	sig.LockTime = 11
	sig.UtxoSequence = 12

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
	require.Equal(t, errInvalidUtxoSequence, err)

	api.On("Query", mock.Anything, mock.Anything,
		mock.Anything).Return(&types.Int64{}, nil)
	err = d.Validate(txMsg, pubKey, types.Encode(sig))
	require.Equal(t, errInvalidUtxoSequence, err)

	cryptocli.SetCurrentBlock(11, types.Now().Unix())
	err = d.Validate(txMsg, pubKey, types.Encode(sig))
	require.Equal(t, errInvalidBtcSignature, err)
}
