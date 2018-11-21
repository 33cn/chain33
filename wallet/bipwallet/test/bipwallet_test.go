// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/33cn/chain33/wallet/bipwallet"
)

func TestBipwallet(t *testing.T) {
	/*目前暂时支持这些币种创建地址

	    TypeEther:        "ETH",
		TypeEtherClassic: "ETC",
		TypeBitcoin:      "BTC",
		TypeLitecoin:     "LTC",
		TypeZayedcoin:    "ZEC",
		TypeBty:          "BTY",
		TypeYcc:          "YCC",
	*/
	//bitsize=128 返回12个单词或者汉子，bitsize+32=160  返回15个单词或者汉子，bitszie=256 返回24个单词或者汉子
	mnem, err := bipwallet.NewMnemonicString(0, 160)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("助记词:", mnem)
	//选择币种，填入种子创建wallet对象
	wallet, err := bipwallet.NewWalletFromMnemonic(bipwallet.TypeEther,
		"wish address cram damp very indicate regret sound figure scheme review scout")
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}
	var index uint32
	//通过索引生成Key pair
	priv, pub, err := wallet.NewKeyPair(index)
	fmt.Println("privkey:", hex.EncodeToString(priv))
	fmt.Println("pubkey:", hex.EncodeToString(pub))
	//通过索引生成对应的地址
	address, err := wallet.NewAddress(index)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("address:", address)
	address, err = bipwallet.PubToAddress(bipwallet.TypeEther, pub)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("PubToAddress:", address)

	pub, err = bipwallet.PrivkeyToPub(bipwallet.TypeEther, priv)
	if err != nil {
		fmt.Println("err:", err.Error())
		return
	}

	fmt.Println("PrivToPub:", hex.EncodeToString(pub))

}
