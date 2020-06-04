package bipwallet

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	mnem          = "叛 促 映 的 庆 站 袖 火 赋 仇 徙 酯 完 砖 乐 据 划 明 犯 谓 杂 模 卷 现"
	ed25519Pub    = "039039742b7dc5553ede2cb3bb61b73bdf5df3b21062b1df75109ba045766cb966"
	ed25519Addr   = "1BAxC6qgPszUMbo4dFDFx6drSgGnf9kfu4"
	secp256k1Pub  = "03b2cb62dd207277abcda55523c467edba786db21106446e040fe2d3515053c8e5"
	secp256k1Addr = "17L828pQH9QGaZe1SfoRXqVau8BcyGJVgP"
)

func TestYccEd25519PrivPub(t *testing.T) {
	wallet, err := NewWalletFromMnemonic(TypeYcc, Ed25519Ty, mnem)
	assert.Nil(t, err)
	priv, pub, err := wallet.NewKeyPair(0)
	assert.Nil(t, err)
	assert.Equal(t, hex.EncodeToString(pub), ed25519Pub)

	//test address
	addr, err := PubToAddress(TypeYcc, pub)
	assert.Nil(t, err)
	assert.Equal(t, addr, ed25519Addr)
	tpub, err := PrivkeyToPub(TypeYcc, priv)
	assert.Nil(t, err)
	assert.Equal(t, tpub, pub)
}

func TestYccSecp256k1PrivPub(t *testing.T) {
	wallet, err := NewWalletFromMnemonic(TypeYcc, Secp256K1Ty, mnem)
	assert.Nil(t, err)
	priv, pub, err := wallet.NewKeyPair(0)
	assert.Nil(t, err)
	assert.Equal(t, hex.EncodeToString(pub), secp256k1Pub)

	//test address
	addr, err := PubToAddress(TypeYcc, pub)
	assert.Nil(t, err)
	assert.Equal(t, addr, secp256k1Addr)
	tpub, err := PrivkeyToPub(TypeYcc, priv)
	assert.Nil(t, err)
	assert.Equal(t, tpub, pub)
}
