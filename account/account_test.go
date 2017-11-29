package account

import (
	"testing"

	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
)

func TestAddress(t *testing.T) {
	c, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		t.Error(err)
		return
	}
	key, err := c.GenKey()
	if err != nil {
		t.Error(err)
		return
	}
        t.Logf("%X", key.Bytes())
	addr := PubKeyToAddress(key.PubKey().Bytes())
	t.Log(addr)
}
