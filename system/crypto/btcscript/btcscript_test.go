// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// btcscript_test btc script test example
package btcscript_test

import (
	"testing"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"

	"github.com/33cn/chain33/system/crypto/btcscript"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/require"
)

// 转账到地址
func Test_ExamplePay2PubKey(t *testing.T) {

	d := btcscript.Driver{}
	priv, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)
	// default sign
	sig := priv.Sign(signMsg)
	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Signature: sig.Bytes(),
		Pubkey:    priv.PubKey().Bytes(),
	}

	require.True(t, tx.CheckSign(0))
	tx.Signature.Pubkey = []byte("invalid pub Key")
	require.False(t, tx.CheckSign(0))
	// with pay2pubKey option
	key, pk := btcscript.NewBtcKeyFromBytes(priv.Bytes())
	addr, lockScript, err := btcscript.GetBtcLockScript(
		btcscript.TyPay2PubKey, pk.SerializeCompressed(), btcscript.Chain33BtcParams)
	require.Equal(t, nil, err)

	signOpts := []interface{}{btcscript.WithBtcLockScript(lockScript),
		btcscript.WithBtcPrivateKeys(&btcscript.BtcAddr2Key{Addr: addr.EncodeAddress(), Key: key})}
	sig = priv.Sign(signMsg, signOpts...)
	tx.Signature.Signature = sig.Bytes()
	tx.Signature.Pubkey = priv.PubKey(signOpts...).Bytes()
	require.True(t, tx.CheckSign(0))
}

func Test_ExamplePay2PubKeyHash(t *testing.T) {

	d := btcscript.Driver{}
	priv, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)

	key, pk := btcscript.NewBtcKeyFromBytes(priv.Bytes())
	addr, pkScript, err := btcscript.GetBtcLockScript(
		btcscript.TyPay2PubKeyHash, pk.SerializeCompressed(), btcscript.Chain33BtcParams)
	require.Equal(t, nil, err)

	signOpts := []interface{}{btcscript.WithBtcLockScript(pkScript),
		btcscript.WithBtcPrivateKeys(&btcscript.BtcAddr2Key{Addr: addr.EncodeAddress(), Key: key})}
	sig := priv.Sign(signMsg, signOpts...)
	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Signature: sig.Bytes(),
		Pubkey:    priv.PubKey(signOpts...).Bytes(),
	}
	require.True(t, tx.CheckSign(0))
	// invalid pub key
	tx.Signature.Pubkey = priv.PubKey().Bytes()
	require.False(t, tx.CheckSign(0))
}

// 转账到脚本
func Test_ExamplePay2ScriptHash(t *testing.T) {

	d := btcscript.Driver{}
	priv, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)

	key, pk := btcscript.NewBtcKeyFromBytes(priv.Bytes())
	addr, pkScript, err := btcscript.GetBtcLockScript(
		btcscript.TyPay2PubKeyHash, pk.SerializeCompressed(), btcscript.Chain33BtcParams)
	require.Equal(t, nil, err)

	scriptAddr, lockScript, err := btcscript.GetBtcLockScript(
		btcscript.TyPay2ScriptHash, pkScript, btcscript.Chain33BtcParams)
	require.Equal(t, nil, err)

	signOpts := []interface{}{btcscript.WithBtcLockScript(lockScript),
		btcscript.WithBtcPrivateKeys(&btcscript.BtcAddr2Key{Addr: addr.EncodeAddress(), Key: key}),
		btcscript.WithBtcScripts(&btcscript.BtcAddr2Script{Addr: scriptAddr.EncodeAddress(), Script: pkScript})}
	sig := priv.Sign(signMsg, signOpts...)
	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Signature: sig.Bytes(),
		Pubkey:    priv.PubKey(signOpts...).Bytes(),
	}
	require.True(t, tx.CheckSign(0))
	// invalid pub key
	tx.Signature.Pubkey = priv.PubKey().Bytes()
	require.False(t, tx.CheckSign(0))
}

// 多重签名
func Test_ExampleMultiSig(t *testing.T) {

	d := btcscript.Driver{}
	priv1, err := d.GenKey()
	require.Nil(t, err)
	priv2, err := d.GenKey()
	require.Nil(t, err)
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	tx := util.CreateNoneTx(cfg, nil)
	signMsg := types.Encode(tx)
	key1, pk1 := btcscript.NewBtcKeyFromBytes(priv1.Bytes())
	key2, pk2 := btcscript.NewBtcKeyFromBytes(priv2.Bytes())
	addr1, err := btcutil.NewAddressPubKey(pk1.SerializeCompressed(), btcscript.Chain33BtcParams)
	require.Nil(t, err)

	addr2, err := btcutil.NewAddressPubKey(pk2.SerializeCompressed(), btcscript.Chain33BtcParams)
	require.Nil(t, err)

	pkScript, err := txscript.MultiSigScript([]*btcutil.AddressPubKey{addr1, addr2}, 2)
	require.Nil(t, err)

	scriptAddr, lockScript, err := btcscript.GetBtcLockScript(
		btcscript.TyPay2ScriptHash, pkScript, btcscript.Chain33BtcParams)
	require.Equal(t, nil, err)

	// 只进行其中一个账户签名
	signOpts := []interface{}{btcscript.WithBtcLockScript(lockScript),
		btcscript.WithBtcPrivateKeys(&btcscript.BtcAddr2Key{
			Addr: addr1.EncodeAddress(), Key: key1}),
		btcscript.WithBtcScripts(&btcscript.BtcAddr2Script{
			Addr: scriptAddr.EncodeAddress(), Script: pkScript}),
	}

	sig1 := priv1.Sign(signMsg, signOpts...)

	tx.Signature = &types.Signature{
		Ty:        btcscript.ID,
		Signature: sig1.Bytes(),
		Pubkey:    priv1.PubKey(signOpts...).Bytes(),
	}
	// 只进行了一个账户签名，验证失败
	require.False(t, tx.CheckSign(0))

	// 在地址1基础上进行地址2的签名
	ssig := &btcscript.Signature{}
	err = types.Decode(sig1.Bytes(), ssig)
	require.Nil(t, err)

	signOpts = []interface{}{
		btcscript.WithPreviousSigScript(ssig.UnlockScript), //指定其中一个签名
		btcscript.WithBtcLockScript(lockScript),
		btcscript.WithBtcPrivateKeys(&btcscript.BtcAddr2Key{
			Addr: addr2.EncodeAddress(), Key: key2}),
		btcscript.WithBtcScripts(&btcscript.BtcAddr2Script{
			Addr: scriptAddr.EncodeAddress(), Script: pkScript}),
	}
	sig2 := priv2.Sign(signMsg, signOpts...)
	tx.Signature.Signature = sig2.Bytes()
	tx.Signature.Pubkey = priv2.PubKey(signOpts...).Bytes()
	require.True(t, tx.CheckSign(0))

	// 单步多签，多个私钥同时签名
	signOpts = []interface{}{
		btcscript.WithBtcLockScript(lockScript),
		btcscript.WithBtcPrivateKeys(&btcscript.BtcAddr2Key{
			Addr: addr1.EncodeAddress(), Key: key1}),
		btcscript.WithBtcPrivateKeys(&btcscript.BtcAddr2Key{
			Addr: addr2.EncodeAddress(), Key: key2}),
		btcscript.WithBtcScripts(&btcscript.BtcAddr2Script{
			Addr: scriptAddr.EncodeAddress(), Script: pkScript}),
	}

	sig := priv1.Sign(signMsg, signOpts...)
	tx.Signature.Signature = sig.Bytes()
	require.True(t, tx.CheckSign(0))

	sig = priv2.Sign(signMsg, signOpts...)
	tx.Signature.Signature = sig.Bytes()
	require.True(t, tx.CheckSign(0))

	// invalid pub key
	tx.Signature.Pubkey = priv1.PubKey().Bytes()
	require.False(t, tx.CheckSign(1))
}
