// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bipwallet 比特币改进协议钱包相关定义
package bipwallet

import (
	"errors"
	"strings"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	bip32 "github.com/33cn/chain33/wallet/bipwallet/go-bip32"
	bip39 "github.com/33cn/chain33/wallet/bipwallet/go-bip39"
	bip44 "github.com/33cn/chain33/wallet/bipwallet/go-bip44"
	"github.com/33cn/chain33/wallet/bipwallet/transformer"
	_ "github.com/33cn/chain33/wallet/bipwallet/transformer/btcbase" //register
)

// https://github.com/satoshilabs/slips/blob/master/slip-0044.md
const (
	TypeBitcoin            uint32 = 0x80000000
	TypeLitecoin           uint32 = 0x80000002
	TypeEther              uint32 = 0x8000003c
	TypeEtherClassic       uint32 = 0x8000003d
	TypeFactomFactoids     uint32 = 0x80000083
	TypeFactomEntryCredits uint32 = 0x80000084
	TypeZcash              uint32 = 0x80000085
	TypeAS                 uint32 = 0x80000382
	TypeBty                uint32 = 0x80003333
	TypeYcc                uint32 = 0x80003334
)

// CoinName 币种名称
var CoinName = map[uint32]string{
	TypeEther:        "ETH",
	TypeEtherClassic: "ETC",
	TypeBitcoin:      "BTC",
	TypeLitecoin:     "LTC",
	TypeZcash:        "ZEC",
	TypeAS:           "AS",
	TypeBty:          "BTY",
	TypeYcc:          "YCC",
}

// coinNameType 映射关系
var coinNameType = map[string]uint32{
	"ETH": TypeEther,
	"ETC": TypeEtherClassic,
	"BTC": TypeBitcoin,
	"LTC": TypeLitecoin,
	"ZEC": TypeZcash,
	"AS":  TypeAS,
	"BTY": TypeBty,
	"YCC": TypeYcc,
}

// GetSLIP0044CoinType 获取货币的 CoinType 值
func GetSLIP0044CoinType(name string) uint32 {
	name = strings.ToUpper(name)
	if ty, ok := coinNameType[name]; ok {
		return ty
	}
	log.Error("GetSLIP0044CoinType: " + name + " not exist.")
	return TypeBty
}

// HDWallet 支持BIP-44标准的HD钱包
type HDWallet struct {
	CoinType  uint32
	RootSeed  []byte
	MasterKey *bip32.Key
	KeyType   uint32
}

// NewKeyPair 通过索引生成新的秘钥对
func (w *HDWallet) NewKeyPair(index uint32) (priv, pub []byte, err error) {
	//bip44 标准 32字节私钥
	key, err := bip44.NewKeyFromMasterKey(w.MasterKey, w.CoinType, bip32.FirstHardenedChild, 0, index)
	if err != nil {
		return nil, nil, err
	}
	if w.KeyType == types.SECP256K1 {
		return key.Key, key.PublicKey().Key, err
	}

	edcrypto, err := crypto.Load(crypto.GetName(int(w.KeyType)), -1)
	if err != nil {
		return nil, nil, err
	}
	edkey, err := edcrypto.PrivKeyFromBytes(key.Key[:])
	if err != nil {
		return nil, nil, err
	}

	priv = edkey.Bytes()
	pub = edkey.PubKey().Bytes()

	return
}

// NewAddress 新建地址
func (w *HDWallet) NewAddress(index uint32) (string, error) {
	if cointype, ok := CoinName[w.CoinType]; ok {
		_, pub, err := w.NewKeyPair(index)
		if err != nil {
			return "", err
		}

		trans, err := transformer.New(cointype)
		if err != nil {
			return "", err
		}
		addr, err := trans.PubKeyToAddress(pub)
		if err != nil {
			return "", err
		}
		return addr, nil
	}

	return "", errors.New("cointype no support to create address")

}

// PrivkeyToPub 私钥转换成公钥
func PrivkeyToPub(coinType, keyTy uint32, priv []byte) ([]byte, error) {
	if cointype, ok := CoinName[coinType]; ok {
		trans, err := transformer.New(cointype)
		if err != nil {
			return nil, err
		}
		pub, err := trans.PrivKeyToPub(keyTy, priv)
		if err != nil {
			return nil, err
		}

		return pub, nil

	}
	return nil, errors.New("cointype no support to create address")
}

// PubToAddress 将公钥转换成地址
func PubToAddress(pub []byte) (string, error) {
	return address.PubKeyToAddr(address.DefaultID, pub), nil
}

// NewMnemonicString 创建助记词 lang=0 英文助记词，lang=1 中文助记词bitsize=[128,256]并且bitsize%32=0
func NewMnemonicString(lang, bitsize int) (string, error) {
	entropy, err := bip39.NewEntropy(bitsize)
	if err != nil {
		return "", err
	}

	mnemonic, err := bip39.NewMnemonic(entropy, int32(lang))
	if err != nil {
		return "", err
	}
	return mnemonic, nil
}

// NewWalletFromMnemonic 通过助记词生成钱包对象
func NewWalletFromMnemonic(coinType, keyType uint32, mnemonic string) (wallet *HDWallet, err error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	return NewWalletFromSeed(coinType, keyType, seed)
}

// NewWalletFromSeed 通过种子生成钱包对象
func NewWalletFromSeed(coinType, keyType uint32, seed []byte) (wallet *HDWallet, err error) {
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}
	return &HDWallet{
		CoinType:  coinType,
		KeyType:   keyType,
		RootSeed:  seed,
		MasterKey: masterKey}, nil
}
