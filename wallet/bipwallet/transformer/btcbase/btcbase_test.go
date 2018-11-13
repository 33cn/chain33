// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package btcbase

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/33cn/chain33/wallet/bipwallet/transformer"
	"github.com/mr-tron/base58/base58"
)

//TODO: 更新USDT的测试数据
var testPrivkey = map[string]string{
	"BTC":  "L3U5kJqHAPXzRGeHvtV5HXTwyjmiAvYG8bk42zswDPz7XwUJ6XfP",
	"BCH":  "L3U5kJqHAPXzRGeHvtV5HXTwyjmiAvYG8bk42zswDPz7XwUJ6XfP",
	"LTC":  "T4qcryHFU8c1kKNWYCFymCafmaVasjm6doJwga5CSDtENenrNwPj",
	"ZEC":  "5HxWvvfubhXpYYpS3tJkw6fq9jE9j18THftkZjHHfmFiWtmAbrj",
	"USDT": "L3U5kJqHAPXzRGeHvtV5HXTwyjmiAvYG8bk42zswDPz7XwUJ6XfP",
	"BTY":  "L3U5kJqHAPXzRGeHvtV5HXTwyjmiAvYG8bk42zswDPz7XwUJ6XfP",
}

var testPrivByte = make(map[string][]byte)

var ansPubkey = map[string]string{
	"BTC":  "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
	"BCH":  "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
	"LTC":  "0325b737cdf14ec8885b578fd901dbb8e2c1020863865656dc74377df4fee67891",
	"ZEC":  "030b4c866585dd868a9d62348a9cd008d6a312937048fff31670e7e920cfc7a744",
	"USDT": "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
	"BTY":  "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
}

var testPubkey = map[string]string{
	"BTC":  "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
	"BCH":  "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
	"LTC":  "0325b737cdf14ec8885b578fd901dbb8e2c1020863865656dc74377df4fee67891",
	"ZEC":  "040b4c866585dd868a9d62348a9cd008d6a312937048fff31670e7e920cfc7a7447b5f0bba9e01e6fe4735c8383e6e7a3347a0fd72381b8f797a19f694054e5a69",
	"USDT": "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
	"BTY":  "0367476225d991b4850b64f751bdb58a65904b70dd09f6cb30f31855f45302ac6a",
}

var ansAddress = map[string]string{
	"BTC":  "1MeionVMkdVuPV82BAXSZsHyxtXNxLFVN8",
	"BCH":  "1MeionVMkdVuPV82BAXSZsHyxtXNxLFVN8",
	"LTC":  "LTFdikYPHPKBbkgfxcvDN9g8x6jgRCmDrE",
	"ZEC":  "t1h8SqgtM3QM5e2M8EzhhT1yL2PXXtA6oqe",
	"USDT": "1MeionVMkdVuPV82BAXSZsHyxtXNxLFVN8",
	"BTY":  "1MeionVMkdVuPV82BAXSZsHyxtXNxLFVN8",
}

//测试基于BTC规则的币种
func TestBtcBaseTransformer(t *testing.T) {
	initialBTCPrivByte()
	for name := range testPrivkey {
		testPrivToPub(t, name)
		testPubToAddr(t, name)
	}
}

//将base58编码的私钥初始化成32字节的形式
func initialBTCPrivByte() (err error) {
	for name, privKey := range testPrivkey {
		privByte, err := base58.Decode(privKey)
		if err != nil {
			return fmt.Errorf("initial priv byte error: %s", err)
		}
		testPrivByte[name] = privByte[1:33]
	}
	return nil
}

//测试私钥生成公钥
func testPrivToPub(t *testing.T, name string) {
	coinTrans, err := transformer.New(name)
	if err != nil {
		t.Errorf("new %s transformer error: %s", name, err)
	}
	pubByte, err := coinTrans.PrivKeyToPub(testPrivByte[name])
	if err != nil {
		t.Errorf("%s PrivKeyToPub error: %s", name, err)
	}
	if hex.EncodeToString(pubByte) == ansPubkey[name] {
		t.Logf("%s public key match", name)
	} else {
		t.Errorf("%s public key mismatch: want: %s have: %x", name, ansPubkey[name], pubByte)
	}
}

//测试公钥生成地址
func testPubToAddr(t *testing.T, name string) {
	coinTrans, err := transformer.New(name)
	if err != nil {
		t.Errorf("new %s transformer error: %s", name, err)
	}
	pubByte, err := hex.DecodeString(testPubkey[name])
	if err != nil {
		t.Errorf("generate %s public key byte error: %s", name, err)
	}
	genAddr, err := coinTrans.PubKeyToAddress(pubByte)
	if err != nil {
		t.Errorf("%s PubKeyToAddress error: %s", name, err)
	}
	if genAddr == ansAddress[name] {
		t.Logf("%s address match", name)
	} else {
		t.Errorf("%s address mismatch: want: %s have %s", name, ansAddress[name], genAddr)
	}
}
