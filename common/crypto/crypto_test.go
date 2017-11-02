package crypto_test

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"testing"

	"code.aliyun.com/chain33/chain33/common/crypto"
	_ "code.aliyun.com/chain33/chain33/common/crypto/ed25519"
	"code.aliyun.com/chain33/chain33/common/crypto/gosm2"
	_ "code.aliyun.com/chain33/chain33/common/crypto/secp256k1"
	_ "code.aliyun.com/chain33/chain33/common/crypto/sm2"
	"github.com/stretchr/testify/require"
)

func TestAll(t *testing.T) {
	testCrypto(t, "ed25519")
	testFromBytes(t, "ed25519")

	testCrypto(t, "secp256k1")
	testFromBytes(t, "secp256k1")

	testCrypto(t, "sm2")
	testFromBytes(t, "sm2")
}

func TestUshieldSm2(t *testing.T) {
	pubKeyStr := "0477dac51899bbdf4fdf3b7b9ba7f931f1e03cc2718894ae20cb823ecc9e5b5c2a170f4345b75c591b13c810d812bd7bd00d42f275cac6fb2ce9546eda3f2d25b3"
	signBase64Str := "Pt3FsCjEGW/Vu57PrQsMkzQhEmoYPDmvgdvg45G6qJhce8iWzkroGkS+yUwUk6cZ1QS85gdjgMmzb6Djqj8Q2w=="
	// 226 bytes
	msg := `<?xml version="1.0" encoding="utf-8"?><T><D><M><k>Remitter Name:</k><v>Alan</v></M><M><k>Remitter Account:</k><v>1234567890123456</v></M><M><k>Amount:</k><v>123.23</v></M></D><E><M><k>Trade ID:</k><v>1234567890</v></M></E></T>`

	require := require.New(t)

	pubBytes, err := hex.DecodeString(pubKeyStr)
	require.Nil(err)

	signBytes, err := base64.StdEncoding.DecodeString(signBase64Str)
	require.Nil(err)

	// 3eddc5b028c4196fd5bb9ecfad0b0c933421126a183c39af81dbe0e391baa8985c7bc896ce4ae81a44bec94c1493a719d504bce6076380c9b36fa0e3aa3f10db
	t.Logf("sm2 sign: %v, %x\n", len(signBytes), signBytes)
	ok := Verify("sm2", pubBytes, []byte(msg), signBytes)
	require.Equal(true, ok)
}

func TestSm3Hash(t *testing.T) {
	msg := "abcd"
	trueVal := "82ec580fe6d36ae4f81cae3c73f4a5b3b5a09c943172dc9053c69fd8e18dca1e"

	require := require.New(t)
	trueBytes, err := hex.DecodeString(trueVal)
	require.Nil(err)

	signBytes := crypto.Sm3Hash([]byte(msg))
	t.Logf("sm3 hash1: %v, %x\n", len(signBytes), signBytes)

	ok := bytes.Equal(signBytes, trueBytes)
	require.Equal(true, ok)
}

func TestSm3Hash2(t *testing.T) {
	msg := "abcd"
	trueVal := "82ec580fe6d36ae4f81cae3c73f4a5b3b5a09c943172dc9053c69fd8e18dca1e"

	require := require.New(t)
	trueBytes, err := hex.DecodeString(trueVal)
	require.Nil(err)

	signBytes, err := gosm2.Hash2([]byte(msg))
	require.Nil(err)
	t.Logf("sm3 hash2: %v, %x\n", len(signBytes), signBytes)

	ok := bytes.Equal(signBytes, trueBytes)
	require.Equal(true, ok)
}

func TestSm3ToSm2(t *testing.T) {
	pubKeyStr := "04F57D96DFACBB90878E768906671DF2F3DA0BD75318C6FE222176EC2EC31BB050D72676DCDFF2BB21198CA42EB81CB09276A515D406CA5F4852572304F3C94865"
	sm3Str := "5D60E23C9FE29B5E62517E144AD67541C6EB132C8926637B6393FE8D9B62B3BF"
	sm2Str := "A4A5ADD2C8F17513BC53F35B7A5ADE30784A26A98B9279B60C66FE4E3D95B50E87DC035DB3F5A14542C852A36E189EFC7039DD6BE41840CC0024BB264071B993"

	require := require.New(t)

	var pubkey gosm2.PublicKey
	err := pubkey.DecodeString(pubKeyStr)
	require.Nil(err)

	hash, err := hex.DecodeString(sm3Str)
	require.Nil(err)

	sig, err := hex.DecodeString(sm2Str)
	require.Nil(err)

	t.Log("sig1:", len(sig))

	sig2, err := gosm2.Sigbin2Der(sig)
	require.Nil(err)

	t.Log("sig2:", len(sig2))

	err = gosm2.Verify(&pubkey, hash, sig)
	err2 := gosm2.Verify(&pubkey, hash, sig2)
	t.Log(err, err2)
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

func Verify(signType string, pubBytes, msg, signBytes []byte) bool {
	c, err := crypto.New(signType)
	if err != nil {
		panic(err)
	}

	pub, err := c.PubKeyFromBytes(pubBytes)
	if err != nil {
		panic(err)
	}

	sign, err := c.SignatureFromBytes(signBytes)
	if err != nil {
		panic(err)
	}

	return pub.VerifyBytes(msg, sign)
}
