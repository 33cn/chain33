package secp256k1sha3

import (
	"strings"
	"testing"

	"github.com/33cn/chain33/common/address"
	_ "github.com/33cn/chain33/system/address"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func Test_All(t *testing.T) {
	testCrypto(t)
	testFromBytes(t)
}

func testCrypto(t *testing.T) {
	require := require.New(t)
	c := &Driver{}
	priv, err := c.GenKey()
	t.Log("privkey", common.Bytes2Hex(priv.Bytes()))
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
	var key = ""

	privKeyBytes := new([privKeyBytesLen]byte)
	copy(privKeyBytes[:], common.FromHex(key))
	var xpriv = PrivKeySecp256k1Sha3(*privKeyBytes)
	hextx := ""

	sig := xpriv.Sign(common.FromHex(hextx))
	t.Log("sigxxxx", common.Bytes2Hex(sig.Bytes()))
	pub = xpriv.PubKey()
	ok = pub.VerifyBytes(common.FromHex(hextx), sig)
	t.Log("publen:", len(pub.Bytes()))
	addr := address.PubKeyToAddr(0, pub.Bytes())

	t.Log("okkkkkkk:", ok, "addr:0", addr)

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
	t.Log("pub str:", common.Bytes2Hex(pub.Bytes()), "len:", len(pub.Bytes()))
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
