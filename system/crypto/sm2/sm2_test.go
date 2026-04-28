package sm2

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tjfoc/gmsm/sm2"
)

func TestAll(t *testing.T) {
	testCrypto(t)
	testFromBytes(t)
	testCryptoCompress(t)
}

func testFromBytes(t *testing.T) {
	require := require.New(t)

	c := &Driver{}

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

func testCrypto(t *testing.T) {
	require := require.New(t)

	c := &Driver{}

	priv, err := c.GenKey()
	require.Nil(err)
	t.Logf("priv:%X, len:%d", priv.Bytes(), len(priv.Bytes()))

	pub := priv.PubKey()
	require.NotNil(pub)
	t.Logf("pub:%X, len:%d", pub.Bytes(), len(pub.Bytes()))

	msg := []byte("hello world")
	signature := priv.Sign(msg)
	t.Logf("sign:%X, len:%d", signature.Bytes(), len(signature.Bytes()))

	ok := pub.VerifyBytes(msg, signature)
	require.Equal(true, ok)
}

func testCryptoCompress(t *testing.T) {
	require := require.New(t)

	c := &Driver{}

	priv, err := c.GenKey()
	require.Nil(err)
	t.Logf("priv:%X, len:%d", priv.Bytes(), len(priv.Bytes()))

	pub := priv.PubKey()
	require.NotNil(pub)
	t.Logf("pub:%X, len:%d, string:%s", pub.Bytes(), len(pub.Bytes()), pub.KeyString())

	pubkey := sm2.Decompress(pub.Bytes())

	pubbytes := SerializePublicKey(pubkey, true)
	pub2, err := c.PubKeyFromBytes(pubbytes)
	assert.Nil(t, err)

	pubbytes = SerializePublicKey(pubkey, false)
	_, err = c.PubKeyFromBytes(pubbytes)
	assert.Nil(t, err)

	msg := []byte("hello world")
	signature := priv.Sign(msg)
	t.Logf("sign:%X, len:%d, string:%s", signature.Bytes(), len(signature.Bytes()), signature.String())

	ok := pub.VerifyBytes(msg, signature)
	require.Equal(true, ok)

	assert.True(t, pub2.VerifyBytes(msg, signature))
	assert.Nil(t, c.Validate(msg, pub.Bytes(), signature.Bytes()))
}

func TestSignatureIsZero(t *testing.T) {
	var sig SignatureSM2
	assert.True(t, sig.IsZero())

	sig = SignatureSM2{1}
	assert.False(t, sig.IsZero())
}

func TestSignatureString(t *testing.T) {
	sig := SignatureSM2{1, 2, 3}
	assert.NotEmpty(t, sig.String())
}

func TestSignatureEquals(t *testing.T) {
	s1 := SignatureSM2{1, 2, 3}
	s2 := SignatureSM2{1, 2, 3}
	s3 := SignatureSM2{4, 5, 6}
	assert.True(t, s1.Equals(s2))
	assert.False(t, s1.Equals(s3))
}

func TestPrivKeyEquals(t *testing.T) {
	p1 := PrivKeySM2{1, 2, 3}
	p2 := PrivKeySM2{1, 2, 3}
	p3 := PrivKeySM2{4, 5, 6}
	assert.True(t, p1.Equals(p2))
	assert.False(t, p1.Equals(p3))
}

func TestPubKeyString(t *testing.T) {
	pub := PubKeySM2{1, 2, 3}
	assert.NotEmpty(t, pub.String())
	assert.NotEmpty(t, pub.KeyString())
}
