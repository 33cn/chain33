// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/33cn/chain33/system/crypto/ed25519"
	"github.com/33cn/chain33/system/crypto/none"
	"github.com/33cn/chain33/system/crypto/secp256k1"

	"github.com/33cn/chain33/common/crypto"
	_ "github.com/33cn/chain33/system/crypto/init"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	require := require.New(t)

	name := crypto.GetName(1)
	require.Equal("secp256k1", name)
	name = crypto.GetName(2)
	require.Equal("ed25519", name)
	name = crypto.GetName(258)
	require.Equal("sm2", name)

	ty := crypto.GetType("secp256k1")
	require.True(ty == 1)
	ty = crypto.GetType("ed25519")
	require.True(ty == 2)

	ty = crypto.GetType("sm2")
	require.True(ty == 258)
}

func TestRipemd160(t *testing.T) {
	b := crypto.Ripemd160([]byte("test"))
	require.NotNil(t, b)
}

func TestSm3Hash(t *testing.T) {
	require := require.New(t)
	b := crypto.Sm3Hash([]byte("test"))
	require.NotNil(b)
}

func TestAll(t *testing.T) {
	testCrypto(t, "ed25519")
	testFromBytes(t, "ed25519")
	testCrypto(t, "secp256k1")
	testFromBytes(t, "secp256k1")
	testCrypto(t, "sm2")
	testFromBytes(t, "sm2")
	testCrypto(t, "secp256r1")
	testFromBytes(t, "secp256r1")
}

func testFromBytes(t *testing.T, name string) {
	require := require.New(t)

	c, err := crypto.New(name)
	require.Nil(err)

	priv, err := c.GenKey()
	require.Nil(err)

	priv2, err := c.PrivKeyFromBytes(priv.Bytes())
	require.Nil(err)
	require.Equal(true, priv.Equals(priv2))

	s1 := string(priv.Bytes())
	s2 := string(priv2.Bytes())
	require.Equal(0, strings.Compare(s1, s2))

	pub := priv.PubKey()
	require.NotNil(pub)

	pub2, err := c.PubKeyFromBytes(pub.Bytes())
	require.Nil(err)
	require.Equal(true, pub.Equals(pub2))

	s1 = string(pub.Bytes())
	s2 = string(pub2.Bytes())
	require.Equal(0, strings.Compare(s1, s2))

	var msg = []byte("hello world")
	sign1 := priv.Sign(msg)
	sign2 := priv2.Sign(msg)

	sign3, err := c.SignatureFromBytes(sign1.Bytes())
	require.Nil(err)
	require.Equal(true, sign3.Equals(sign1))

	require.Equal(true, pub.VerifyBytes(msg, sign1))
	require.Equal(true, pub2.VerifyBytes(msg, sign1))
	require.Equal(true, pub.VerifyBytes(msg, sign2))
	require.Equal(true, pub2.VerifyBytes(msg, sign2))
	require.Equal(true, pub.VerifyBytes(msg, sign3))
	require.Equal(true, pub2.VerifyBytes(msg, sign3))
	require.Nil(c.Validate(msg, pub.Bytes(), sign1.Bytes()))
}

func testCrypto(t *testing.T, name string) {
	require := require.New(t)

	c, err := crypto.New(name)
	require.Nil(err)

	priv, err := c.GenKey()
	require.Nil(err)
	t.Logf("%s priv:%X, len:%d", name, priv.Bytes(), len(priv.Bytes()))

	pub := priv.PubKey()
	require.NotNil(pub)
	t.Logf("%s pub:%X, len:%d", name, pub.Bytes(), len(pub.Bytes()))

	msg := []byte("hello world")
	signature := priv.Sign(msg)
	t.Logf("%s sign:%X, len:%d", name, signature.Bytes(), len(signature.Bytes()))

	ok := pub.VerifyBytes(msg, signature)
	require.Equal(true, ok)
}

func BenchmarkSignEd25519(b *testing.B) {
	benchSign(b, "ed25519")
}

func BenchmarkVerifyEd25519(b *testing.B) {
	benchVerify(b, "ed25519")
}

func BenchmarkSignSecp256k1(b *testing.B) {
	benchSign(b, "secp256k1")
}

func BenchmarkVerifySecp256k1(b *testing.B) {
	benchVerify(b, "secp256k1")
}

func BenchmarkSignSm2(b *testing.B) {
	benchSign(b, "sm2")
}

func BenchmarkVerifySm2(b *testing.B) {
	benchVerify(b, "sm2")
}

func benchSign(b *testing.B, name string) {
	c, _ := crypto.New(name)
	priv, _ := c.GenKey()
	msg := []byte("hello world")
	for i := 0; i < b.N; i++ {
		priv.Sign(msg)
	}
}

func benchVerify(b *testing.B, name string) {
	c, _ := crypto.New(name)
	priv, _ := c.GenKey()
	pub := priv.PubKey()
	msg := []byte("hello world")
	sign := priv.Sign(msg)
	for i := 0; i < b.N; i++ {
		pub.VerifyBytes(msg, sign)
	}
}

func TestAggregate(t *testing.T) {
	c, err := crypto.New("secp256k1")
	if err != nil {
		panic(err)
	}
	_, err = crypto.ToAggregate(c)
	require.Equal(t, err, crypto.ErrNotSupportAggr)

	c = democrypto{}
	aggr, err := crypto.ToAggregate(c)
	require.Nil(t, err)
	sig, err := aggr.Aggregate(nil)
	require.Nil(t, sig)
	require.Nil(t, err)
}

type democrypto struct{}

func (d democrypto) GenKey() (crypto.PrivKey, error) {
	return nil, nil
}

func (d democrypto) SignatureFromBytes([]byte) (crypto.Signature, error) {
	return nil, nil
}
func (d democrypto) PrivKeyFromBytes([]byte) (crypto.PrivKey, error) {
	return nil, nil
}
func (d democrypto) PubKeyFromBytes([]byte) (crypto.PubKey, error) {
	return nil, nil
}
func (d democrypto) Validate(msg, pub, sig []byte) error {
	return nil
}

//AggregateCrypto 聚合签名

func (d democrypto) Aggregate(sigs []crypto.Signature) (crypto.Signature, error) {
	return nil, nil
}
func (d democrypto) AggregatePublic(pubs []crypto.PubKey) (crypto.PubKey, error) {
	return nil, nil
}
func (d democrypto) VerifyAggregatedOne(pubs []crypto.PubKey, m []byte, sig crypto.Signature) error {
	return nil
}
func (d democrypto) VerifyAggregatedN(pubs []crypto.PubKey, ms [][]byte, sig crypto.Signature) error {
	return nil
}

type democryptoCGO struct {
	democrypto
}

func (d democryptoCGO) GenKey() (crypto.PrivKey, error) {
	return nil, errors.New("testCGO")
}

func TestRegister(t *testing.T) {
	c, err := crypto.New("secp256k1")
	require.Nil(t, err)
	p, err := c.GenKey()
	require.Nil(t, err)
	require.NotNil(t, p)
	require.Panics(t, func() { crypto.Register(secp256k1.Name, democryptoCGO{}, crypto.WithRegOptionTypeID(secp256k1.ID)) })
	//注册cgo版本，替换
	crypto.Register(secp256k1.Name, democryptoCGO{}, crypto.WithRegOptionCGO(), crypto.WithRegOptionTypeID(secp256k1.ID))
	//重复注册非cgo版本，不会报错
	crypto.Register(secp256k1.Name, democryptoCGO{}, crypto.WithRegOptionTypeID(secp256k1.ID))
	require.Panics(t, func() {
		crypto.Register(secp256k1.Name, democryptoCGO{}, crypto.WithRegOptionCGO(), crypto.WithRegOptionTypeID(1024))
	})
	require.Panics(t, func() {
		crypto.Register(secp256k1.Name+"cgo", democryptoCGO{}, crypto.WithRegOptionTypeID(secp256k1.ID))
	})

	c, err = crypto.New("secp256k1")
	require.Nil(t, err)
	p, err = c.GenKey()
	require.Nil(t, p)
	require.Equal(t, errors.New("testCGO"), err)
}

func getNewCryptoErr(name string, height int64) error {
	_, err := crypto.New(name, crypto.WithNewOptionEnableCheck(height))
	return err
}

func TestInitCfg(t *testing.T) {

	cfg := &crypto.Config{}
	cfg.EnableHeight = make(map[string]int64)
	cfg.EnableHeight[none.Name] = 0
	crypto.Init(cfg, nil)
	must := require.New(t)
	must.NotNil(getNewCryptoErr(none.Name, 0))
	must.Nil(getNewCryptoErr(secp256k1.Name, 0))
	must.Nil(getNewCryptoErr(ed25519.Name, 0))
	cfg.EnableTypes = []string{secp256k1.Name, none.Name}
	cfg.EnableHeight[ed25519.Name] = 10
	cfg.EnableHeight[secp256k1.Name] = -1
	cfg.EnableHeight[none.Name] = 100
	crypto.Init(cfg, nil)
	must.NotNil(getNewCryptoErr(none.Name, 0))
	must.Nil(getNewCryptoErr(none.Name, 100))
	must.NotNil(getNewCryptoErr(secp256k1.Name, -1))
	must.NotNil(getNewCryptoErr(ed25519.Name, 10))
}

type testSubCfg struct {
	Name   string
	Height int64
}

func TestInitSubCfg(t *testing.T) {

	cfg := &crypto.Config{}
	subCfg := make(map[string][]byte)

	sub1 := &testSubCfg{Name: "test", Height: 100}
	bsub, err := json.Marshal(sub1)
	require.Nil(t, err)
	initFn := func(b []byte) {
		sub2 := &testSubCfg{}
		err := json.Unmarshal(b, sub2)
		require.Nil(t, err)
		require.Equal(t, sub1, sub2)
	}
	crypto.Register("test", democrypto{}, crypto.WithRegOptionInitFunc(initFn))
	subCfg[sub1.Name] = bsub
	crypto.Init(cfg, subCfg)
}

func TestGenDriverTypeID(t *testing.T) {
	id := crypto.GenDriverTypeID("TestGenDriverTypeID")
	require.Equal(t, int32(81208513), id)
}

func TestWithOption(t *testing.T) {
	driver := &crypto.Driver{}
	option := crypto.WithRegOptionTypeID(-1)
	require.NotNil(t, option(driver))
	option = crypto.WithRegOptionTypeID(crypto.MaxManualTypeID)
	require.Nil(t, option(driver))
	option = crypto.WithRegOptionTypeID(crypto.MaxManualTypeID + 1)
	require.NotNil(t, option(driver))
	option = crypto.WithRegOptionInitFunc(nil)
	require.NotNil(t, option(driver))
}
