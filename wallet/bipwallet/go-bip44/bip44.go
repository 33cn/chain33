// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Copyright 2016 Factom Foundation
// Use of this source code is governed by the MIT
// license that can be found in the LICENSE file.

// Package bip44 基于 BIP32 的系统，赋予树状结构中的各层特殊的意义。让同一个 seed 可以支援多币种、多帐户等。
package bip44

import (
	bip32 "github.com/33cn/chain33/wallet/bipwallet/go-bip32"
	bip39 "github.com/33cn/chain33/wallet/bipwallet/go-bip39"
)

// Purpose Purpose
const Purpose uint32 = 0x8000002C

//https://github.com/bitcoin/bips/blob/master/bip-0044.mediawiki
//https://github.com/satoshilabs/slips/blob/master/slip-0044.md
//https://github.com/FactomProject/FactomDocs/blob/master/wallet_info/wallet_test_vectors.md

/*
const (
	// TypeBitcoin 比特币类型
	TypeBitcoin uint32 = 0x80000000
	TypeTestnet uint32 = 0x80000001
	// TypeLitecoin 莱特币类型
	TypeLitecoin              uint32 = 0x80000002
	TypeDogecoin              uint32 = 0x80000003
	TypeReddcoin              uint32 = 0x80000004
	TypeDash                  uint32 = 0x80000005
	TypePeercoin              uint32 = 0x80000006
	TypeNamecoin              uint32 = 0x80000007
	TypeFeathercoin           uint32 = 0x80000008
	TypeCounterparty          uint32 = 0x80000009
	TypeBlackcoin             uint32 = 0x8000000a
	TypeNuShares              uint32 = 0x8000000b
	TypeNuBits                uint32 = 0x8000000c
	TypeMazacoin              uint32 = 0x8000000d
	TypeViacoin               uint32 = 0x8000000e
	TypeClearingHouse         uint32 = 0x8000000f
	TypeRubycoin              uint32 = 0x80000010
	TypeGroestlcoin           uint32 = 0x80000011
	TypeDigitalcoin           uint32 = 0x80000012
	TypeCannacoin             uint32 = 0x80000013
	TypeDigiByte              uint32 = 0x80000014
	TypeOpenAssets            uint32 = 0x80000015
	TypeMonacoin              uint32 = 0x80000016
	TypeClams                 uint32 = 0x80000017
	TypePrimecoin             uint32 = 0x80000018
	TypeNeoscoin              uint32 = 0x80000019
	TypeJumbucks              uint32 = 0x8000001a
	TypeziftrCOIN             uint32 = 0x8000001b
	TypeVertcoin              uint32 = 0x8000001c
	TypeNXT                   uint32 = 0x8000001d
	TypeBurst                 uint32 = 0x8000001e
	TypeMonetaryUnit          uint32 = 0x8000001f
	TypeZoom                  uint32 = 0x80000020
	TypeVpncoin               uint32 = 0x80000021
	TypeCanadaeCoin           uint32 = 0x80000022
	TypeShadowCash            uint32 = 0x80000023
	TypeParkByte              uint32 = 0x80000024
	TypePandacoin             uint32 = 0x80000025
	TypeStartCOIN             uint32 = 0x80000026
	TypeMOIN                  uint32 = 0x80000027
	TypeArgentum              uint32 = 0x8000002D
	TypeGlobalCurrencyReserve uint32 = 0x80000031
	TypeNovacoin              uint32 = 0x80000032
	TypeAsiacoin              uint32 = 0x80000033
	TypeBitcoindark           uint32 = 0x80000034
	TypeDopecoin              uint32 = 0x80000035
	TypeTemplecoin            uint32 = 0x80000036
	TypeAIB                   uint32 = 0x80000037
	TypeEDRCoin               uint32 = 0x80000038
	TypeSyscoin               uint32 = 0x80000039
	TypeSolarcoin             uint32 = 0x8000003a
	TypeSmileycoin            uint32 = 0x8000003b
	// TypeEther 以太坊类型
	TypeEther uint32 = 0x8000003c
	// TypeEtherClassic 经典以太坊
	TypeEtherClassic     uint32 = 0x8000003d
	TypeOpenChain        uint32 = 0x80000040
	TypeOKCash           uint32 = 0x80000045
	TypeDogecoinDark     uint32 = 0x8000004d
	TypeElectronicGulden uint32 = 0x8000004e
	TypeClubCoin         uint32 = 0x8000004f
	TypeRichCoin         uint32 = 0x80000050
	TypePotcoin          uint32 = 0x80000051
	TypeQuarkcoin        uint32 = 0x80000052
	TypeTerracoin        uint32 = 0x80000053
	TypeGridcoin         uint32 = 0x80000054
	TypeAuroracoin       uint32 = 0x80000055
	TypeIXCoin           uint32 = 0x80000056
	TypeGulden           uint32 = 0x80000057
	TypeBitBean          uint32 = 0x80000058
	TypeBata             uint32 = 0x80000059
	TypeMyriadcoin       uint32 = 0x8000005a
	TypeBitSend          uint32 = 0x8000005b
	TypeUnobtanium       uint32 = 0x8000005c
	TypeMasterTrader     uint32 = 0x8000005d
	TypeGoldBlocks       uint32 = 0x8000005e
	TypeSaham            uint32 = 0x8000005f
	TypeChronos          uint32 = 0x80000060
	TypeUbiquoin         uint32 = 0x80000061
	TypeEvotion          uint32 = 0x80000062
	TypeSaveTheOcean     uint32 = 0x80000063
	TypeBigUp            uint32 = 0x80000064
	TypeGameCredits      uint32 = 0x80000065
	TypeDollarcoins      uint32 = 0x80000066
	// TypeZayedcoin Zayed币
	TypeZayedcoin uint32 = 0x80000067
	TypeDubaicoin uint32 = 0x80000068
	TypeStratis   uint32 = 0x80000069
	TypeShilling  uint32 = 0x8000006a
	TypePiggyCoin uint32 = 0x80000076
	TypeMonero    uint32 = 0x80000080
	TypeNavCoin   uint32 = 0x80000082
	// TypeFactomFactoids FactomFactoids
	TypeFactomFactoids uint32 = 0x80000083
	// TypeFactomEntryCredits FactomEntryCredits
	TypeFactomEntryCredits uint32 = 0x80000084
	TypeZcash              uint32 = 0x80000085
	TypeLisk               uint32 = 0x80000086
)
*/

// NewKeyFromMnemonic 新建Key
func NewKeyFromMnemonic(mnemonic string, coin, account, chain, address uint32) (*bip32.Key, error) {
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return nil, err
	}

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, err
	}

	return NewKeyFromMasterKey(masterKey, coin, account, chain, address)
}

// NewKeyFromMasterKey 新建Key
func NewKeyFromMasterKey(masterKey *bip32.Key, coin, account, chain, address uint32) (*bip32.Key, error) {
	child, err := masterKey.NewChildKey(Purpose)
	if err != nil {
		return nil, err
	}

	child, err = child.NewChildKey(coin)
	if err != nil {
		return nil, err
	}

	child, err = child.NewChildKey(account)
	if err != nil {
		return nil, err
	}

	child, err = child.NewChildKey(chain)
	if err != nil {
		return nil, err
	}

	child, err = child.NewChildKey(address)
	if err != nil {
		return nil, err
	}

	return child, nil
}
