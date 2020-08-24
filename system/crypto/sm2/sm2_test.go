package sm2

import (
	"strings"
	"testing"

	"github.com/33cn/chain33/common/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tjfoc/gmsm/sm2"
)

func TestAll(t *testing.T) {
	testCrypto(t, "sm2")
	testFromBytes(t, "sm2")
	testCryptoUncompress(t, "sm2")
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

func testCryptoUncompress(t *testing.T, name string) {
	require := require.New(t)

	c, err := crypto.New(name)
	require.Nil(err)

	priv, err := c.GenKey()
	require.Nil(err)
	t.Logf("%s priv:%X, len:%d", name, priv.Bytes(), len(priv.Bytes()))

	pub := priv.PubKey()
	require.NotNil(pub)
	t.Logf("%s pub:%X, len:%d", name, pub.Bytes(), len(pub.Bytes()))

	pubkey := sm2.Decompress(pub.Bytes())

	pubbytes := SerializePublicKey(pubkey, true)
	pub2, err := c.PubKeyFromBytes(pubbytes)
	assert.Nil(t, err)

	msg := []byte("hello world")
	signature := priv.Sign(msg)
	t.Logf("%s sign:%X, len:%d", name, signature.Bytes(), len(signature.Bytes()))

	ok := pub.VerifyBytes(msg, signature)
	require.Equal(true, ok)

	assert.True(t, pub2.VerifyBytes(msg, signature))
}
