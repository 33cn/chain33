package ed25519

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/33cn/chain33/common"
	"github.com/stretchr/testify/assert"
)

func TestGenKey(t *testing.T) {
	d := &Driver{}
	key, err := d.GenKey()
	assert.Nil(t, err)
	assert.Equal(t, 64, len(key.Bytes()))
}

func TestPrivateKeyFromBytes(t *testing.T) {
	hexstr := "99827c6a9f0aae804ab0a0a3676c6502464864fd8398e0346895e6996ac77a81ae67c2c99ce7590c3010f9f813c517f2261ccd18cde910f13ec00caaa3360ff1"
	assert.Len(t, hexstr, 128)

	pbytes, err := hex.DecodeString(hexstr)
	assert.Nil(t, err)
	d := &Driver{}
	priv, err := d.PrivKeyFromBytes(pbytes)
	assert.Nil(t, err)
	assert.Equal(t, pbytes, priv.Bytes())

	priv, err = d.PrivKeyFromBytes(pbytes[0:32])
	assert.Nil(t, err)
	assert.Equal(t, pbytes, priv.Bytes())

	assert.Equal(t, priv.PubKey().Bytes(), pbytes[32:])
	assert.True(t, priv.Equals(priv))

	//sig
	msg := []byte("message")
	sig := priv.Sign(msg)
	sighex := fmt.Sprintf("%X", sig.Bytes())
	assert.Equal(t, "/"+sighex+".../", sig.String())

	sigbytes, err := common.FromHex(sighex)
	assert.Nil(t, err)
	sig2, err := d.SignatureFromBytes(sigbytes)
	assert.Nil(t, err)
	assert.True(t, sig2.Equals(sig))
	assert.False(t, sig2.IsZero())

	//pub
	pub := priv.PubKey()
	pubhex := fmt.Sprintf("%X", pub.Bytes())
	assert.Equal(t, pubhex, pub.KeyString())
	pub2, err := d.PubKeyFromBytes(pub.Bytes())
	assert.Nil(t, err)
	assert.True(t, pub.Equals(pub2))

	assert.True(t, pub2.VerifyBytes(msg, sig))
	assert.Nil(t, d.Validate(msg, pub.Bytes(), sigbytes))
}
