package crypto_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	_ "gitlab.33.cn/chain33/chain33/system/crypto/init"
)

func TestAll(t *testing.T) {
	testCrypto(t, "ed25519")
	testFromBytes(t, "ed25519")
	testCrypto(t, "secp256k1")
	testFromBytes(t, "secp256k1")
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
