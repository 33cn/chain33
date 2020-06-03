package bipwallet

import (
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	mnem   = "叛 促 映 的 庆 站 袖 火 赋 仇 徙 酯 完 砖 乐 据 划 明 犯 谓 杂 模 卷 现"
	pubstr = "03e455b9e322dd2f51a7c0a9aeac08285c96d37b46fbe6762669b9671f1aafb7e0"
	taddr  = "1BYSZKNMUFdyieML9Cw5iwGPAdyQ3oYNS7"
)

func TestYccPrivPub(t *testing.T) {
	wallet, err := NewWalletFromMnemonic(TypeYcc, mnem)
	assert.Nil(t, err)
	priv, pub, err := wallet.NewKeyPair(0)
	assert.Nil(t, err)
	assert.Equal(t, hex.EncodeToString(pub), pubstr)

	//test address
	addr, err := PubToAddress(TypeYcc, pub)
	assert.Nil(t, err)
	assert.Equal(t, addr, taddr)
	tpub, err := PrivkeyToPub(TypeYcc, priv)
	assert.Nil(t, err)
	assert.Equal(t, tpub, pub)
}
