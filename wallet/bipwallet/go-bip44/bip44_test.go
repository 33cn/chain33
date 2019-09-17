// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bip44_test

import (
	"testing"

	"github.com/33cn/chain33/wallet/bipwallet"
	bip32 "github.com/33cn/chain33/wallet/bipwallet/go-bip32"
	bip39 "github.com/33cn/chain33/wallet/bipwallet/go-bip39"
	. "github.com/33cn/chain33/wallet/bipwallet/go-bip44"
	"github.com/stretchr/testify/assert"
)

func TestNewKeyFromMnemonic(t *testing.T) {
	mnemonic := "yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow"
	fKey, err := NewKeyFromMnemonic(mnemonic, bipwallet.TypeFactomFactoids, bip32.FirstHardenedChild, 0, 0)
	if err != nil {
		panic(err)
	}
	if fKey.String() != "xprvA2vH8KdcBBKhMxhENJpJdbwiU5cUXSkaHR7QVTpBmusgYMR8NsZ4BFTNyRLUiaPHg7UYP8u92FJkSEAmmgu3PDQCoY7gBsdvpB7msWGCpXG" {
		t.Errorf("Invalid Factoid key - %v", fKey.String())
	}

	ecKey, err := NewKeyFromMnemonic(mnemonic, bipwallet.TypeFactomEntryCredits, bip32.FirstHardenedChild, 0, 0)
	if err != nil {
		panic(err)
	}
	if ecKey.String() != "xprvA2ziNegvZRfAAUtDsjeS9LvCP1TFXfR3hUzMcJw7oYAhdPqZyiJTMf1ByyLRxvQmGvgbPcG6Q569m26ixWsmgTR3d3PwicrG7hGD7C7seJA" {
		t.Errorf("Invalid EC key - %v", ecKey.String())
	}

	mnemonic = "aaaaaa"
	_, err = NewKeyFromMnemonic(mnemonic, bipwallet.TypeFactomFactoids, bip32.FirstHardenedChild, 0, 0)
	assert.NotNil(t, err)
}

func TestNewKeyFromMasterKey(t *testing.T) {
	mnemonic := "yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow"

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		panic(err)
	}

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		panic(err)
	}

	fKey, err := NewKeyFromMasterKey(masterKey, bipwallet.TypeFactomFactoids, bip32.FirstHardenedChild, 0, 0)
	if err != nil {
		panic(err)
	}
	if fKey.String() != "xprvA2vH8KdcBBKhMxhENJpJdbwiU5cUXSkaHR7QVTpBmusgYMR8NsZ4BFTNyRLUiaPHg7UYP8u92FJkSEAmmgu3PDQCoY7gBsdvpB7msWGCpXG" {
		t.Errorf("Invalid Factoid key - %v", fKey.String())
	}

	ecKey, err := NewKeyFromMasterKey(masterKey, bipwallet.TypeFactomEntryCredits, bip32.FirstHardenedChild, 0, 0)
	if err != nil {
		panic(err)
	}
	if ecKey.String() != "xprvA2ziNegvZRfAAUtDsjeS9LvCP1TFXfR3hUzMcJw7oYAhdPqZyiJTMf1ByyLRxvQmGvgbPcG6Q569m26ixWsmgTR3d3PwicrG7hGD7C7seJA" {
		t.Errorf("Invalid EC key - %v", ecKey.String())
	}

	_, err = NewKeyFromMasterKey(&bip32.Key{}, bipwallet.TypeFactomFactoids, bip32.FirstHardenedChild, 0, 0)
	assert.NotNil(t, err)
}

/*
func TestTest(t *testing.T) {
	//var factoidHex uint32 = 0x80000083
	//var ecHex uint32 = 0x80000084

	mnemonic := "yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow yellow"

	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		panic(err)
	}

	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		panic(err)
	}

	child, err := masterKey.NewChildKey(bip32.FirstHardenedChild + 44)
	if err != nil {
		panic(err)
	}
	t.Logf("%v", child.String())

	child, err = child.NewChildKey(bip32.FirstHardenedChild + 132)
	if err != nil {
		panic(err)
	}
	t.Logf("%v", child.String())

	child, err = child.NewChildKey(bip32.FirstHardenedChild)
	if err != nil {
		panic(err)
	}
	t.Logf("%v", child.String())

	child, err = child.NewChildKey(0)
	if err != nil {
		panic(err)
	}
	t.Logf("%v", child.String())

	child, err = child.NewChildKey(0)
	if err != nil {
		panic(err)
	}
	t.Logf("%v", child.String())

	/*
		if child.String()!="xprvA22bpQKA9av7gEKdskwxbBNaMso6XpmW7sXi5LGgKnGCMe82BYW68tcNXtn4ZiLHDYJ2HpRvknV7zdDSgBXtPo4dRwG8XCcU55akAcarx3G" {

		}
*/ /*

	key, err := NewKeyFromMnemonic(mnemonic, bip32.FirstHardenedChild, 0, 0, 0)
	if err != nil {
		panic(err)
	}
	t.Logf("%v", key.String())

	add, err := factom.MakeFactoidAddress(key.Key)
	if err != nil {
		panic(err)
	}
	t.Logf("%v", add.String())

	t.FailNow()
}
*/
