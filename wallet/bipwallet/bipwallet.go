// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package bipwallet 比特币改进协议钱包相关定义
package bipwallet

import (
	"errors"
	"github.com/NebulousLabs/Sia/crypto"

	bip32 "github.com/33cn/chain33/wallet/bipwallet/go-bip32"
	bip39 "github.com/33cn/chain33/wallet/bipwallet/go-bip39"
	bip44 "github.com/33cn/chain33/wallet/bipwallet/go-bip44"
	"github.com/33cn/chain33/wallet/bipwallet/transformer"
	_ "github.com/33cn/chain33/wallet/bipwallet/transformer/btcbase" //register btcbase package
	_ "github.com/33cn/chain33/wallet/bipwallet/transformer/ed25519base"
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
	TypeBty:          "BTY",
	TypeYcc:          "YCC",
}

// HDWallet 支持BIP-44标准的HD钱包
type HDWallet struct {
	CoinType  uint32
	RootSeed  []byte
	MasterKey *bip32.Key
}

// NewKeyPair 通过索引生成新的秘钥对
func (w *HDWallet) NewKeyPair(index uint32) (priv, pub []byte, err error) {
	if w.CoinType != TypeYcc { //bip44 标准 32字节私钥
		key, err := bip44.NewKeyFromMasterKey(w.MasterKey, w.CoinType, bip32.FirstHardenedChild, 0, index)
		if err != nil {
			return nil, nil, err
		}
		return key.Key, key.PublicKey().Key, err
	}

	//非bip44标准创建公私钥对,64字节私钥
	return seedToSecretkey(w.RootSeed, index)
}

//通过助记词形式的seed生成私钥和公钥,一个seed根据不同的index可以生成许多组密钥
func seedToSecretkey(seed []byte, index uint32) (secretKey, publicKey []byte, err error) {

	sk, pk := crypto.GenerateKeyPairDeterministic(crypto.HashAll(seed, index))
	var raw []byte = make([]byte, 33)
	raw[0] = 0x03
	copy(raw[1:], pk[:])

	return sk[:], raw, nil
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
func PrivkeyToPub(coinType uint32, priv []byte) ([]byte, error) {
	if cointype, ok := CoinName[coinType]; ok {
		trans, err := transformer.New(cointype)
		if err != nil {
			return nil, err
		}
		pub, err := trans.PrivKeyToPub(priv)
		if err != nil {
			return nil, err
		}

		return pub, nil

	}
	return nil, errors.New("cointype no support to create address")
}

// PubToAddress 将公钥转换成地址
func PubToAddress(coinType uint32, pub []byte) (string, error) {
	if cointype, ok := CoinName[coinType]; ok {
		trans, err := transformer.New(cointype)
		if err != nil {
			return "", err
		}
		pub, err := trans.PubKeyToAddress(pub)
		if err != nil {
			return "", err
		}

		return pub, nil

	}
	return "", errors.New("cointype no support to create address")
}

//NewMnemonicString 创建助记词 lang=0 英文助记词，lang=1 中文助记词bitsize=[128,256]并且bitsize%32=0
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
func NewWalletFromMnemonic(coinType uint32, mnemonic string) (wallet *HDWallet, err error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	return NewWalletFromSeed(coinType, seed)
}

// NewWalletFromSeed 通过种子生成钱包对象
func NewWalletFromSeed(coinType uint32, seed []byte) (wallet *HDWallet, err error) {
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}
	return &HDWallet{
		CoinType:  coinType,
		RootSeed:  seed,
		MasterKey: masterKey}, nil
}
