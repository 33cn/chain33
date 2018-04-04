package account

import (
	"encoding/hex"
	"testing"

	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
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

func TestPubkeyToAddress(t *testing.T) {
	pubkey := "024a17b0c6eb3143839482faa7e917c9b90a8cfe5008dff748789b8cea1a3d08d5"
	b, err := hex.DecodeString(pubkey)
	if err != nil {
		t.Error(err)
		return
	}
	t.Logf("%X", b)
	addr := PubKeyToAddress(b)
	t.Log(addr)
}
