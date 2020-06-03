package ed25519base

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	Tmnem = "叛 促 映 的 庆 站 袖 火 赋 仇 徙 酯 完 砖 乐 据 划 明 犯 谓 杂 模 卷 现"
	pub   = "0342a699365ceccfb7720c5efeb5b7016acf31fd8975ce5e8d450d368393fd5135"
	pub2  = "039cbc600290662184fa77d33fba561c5811a896872510ef6925caa77eb727902e"
	addr  = "1DugwWFs8HoLufWyD74u5JThrtmq7zrVUp"
)

func TestSeedKey(t *testing.T) {
	priv, pubstr, err := MnemonicToSecretkey(Tmnem, 1)
	assert.Nil(t, err)
	t.Log("priv", priv, "size", len(priv), "pub", pub, "pub size", len(pubstr))
	pubbs, _ := hex.DecodeString(pubstr)
	assert.Equal(t, 3, int(pubbs[0]))
	assert.Equal(t, pubstr, pub)
	_, pubstr2, err := MnemonicToSecretkey(Tmnem, 2)
	assert.Nil(t, err)
	assert.Equal(t, pubstr2, pub2)

}

func TestPubToAddress(t *testing.T) {
	pubbs, _ := hex.DecodeString(pub)
	t.Log("pubs", len(pubbs)) //必须33字节
	bt := &baseTransformer{[]byte{0x00}}
	addrstr, err := bt.PubKeyToAddress(pubbs)
	assert.Nil(t, err)
	assert.Equal(t, addrstr, addr)
}

func TestPrivToPub(t *testing.T) {
	bt := &baseTransformer{[]byte{0x00}}
	priv, _, err := MnemonicToSecretkey(Tmnem, 1)
	assert.Nil(t, err)
	privbs, _ := hex.DecodeString(priv)
	pubbs, err := bt.PrivKeyToPub(privbs)
	assert.Nil(t, err)
	assert.Equal(t, hex.EncodeToString(pubbs), pub)

}
