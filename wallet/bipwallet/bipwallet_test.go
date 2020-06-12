package bipwallet

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

var (
	mnem          = "叛 促 映 的 庆 站 袖 火 赋 仇 徙 酯 完 砖 乐 据 划 明 犯 谓 杂 模 卷 现"
	ed25519Pub    = "9039742b7dc5553ede2cb3bb61b73bdf5df3b21062b1df75109ba045766cb966"
	ed25519Addr   = "1Ptp353sEjjFa37UToFkeFRXSkNTXnxNM3"
	secp256k1Pub  = "03b2cb62dd207277abcda55523c467edba786db21106446e040fe2d3515053c8e5"
	secp256k1Addr = "17L828pQH9QGaZe1SfoRXqVau8BcyGJVgP"
	sm2Pub        = "01c4b4e32badcd3cb67bd43cf7d597e993ba2e4b5d121afbf22b4c5f7f064dea97"
	sm2Addr       = "1QEqU8FdqtEGX96tBocaDmicMwEgBu88fH"
)

func TestNewMnemonicString(t *testing.T) {
	mne, err := NewMnemonicString(1, 128)
	assert.Nil(t, err)
	t.Log(mne)
	mne, err = NewMnemonicString(0, 128)
	assert.Nil(t, err)
	t.Log(mne)

	_, err = NewMnemonicString(0, 129)
	assert.NotNil(t, err)
}

func TestNewWalletFromMnemonic(t *testing.T) {
	_, err := NewWalletFromMnemonic(TypeYcc, types.ED25519, mnem+"test")
	assert.NotNil(t, err)
}

func TestEd25519PrivPub(t *testing.T) {
	wallet, err := NewWalletFromMnemonic(TypeYcc, types.ED25519, mnem)
	assert.Nil(t, err)
	priv, pub, err := wallet.NewKeyPair(0)
	assert.Nil(t, err)
	assert.Equal(t, len(priv), 64)
	assert.Equal(t, hex.EncodeToString(pub), ed25519Pub)

	//test address
	addr, err := PubToAddress(pub)
	assert.Nil(t, err)
	assert.Equal(t, addr, ed25519Addr)
	naddr, err := wallet.NewAddress(0)
	assert.Nil(t, err)
	assert.Equal(t, addr, naddr)
	tpub, err := PrivkeyToPub(TypeYcc, types.ED25519, priv)
	assert.Nil(t, err)
	assert.Equal(t, tpub, pub)
	_, err = PrivkeyToPub(TypeYcc+1, types.ED25519, priv)
	assert.NotNil(t, err)

}

func TestSecp256k1PrivPub(t *testing.T) {
	wallet, err := NewWalletFromMnemonic(TypeYcc, types.SECP256K1, mnem)
	assert.Nil(t, err)
	priv, pub, err := wallet.NewKeyPair(0)
	assert.Nil(t, err)
	assert.Equal(t, 32, len(priv))
	assert.Equal(t, hex.EncodeToString(pub), secp256k1Pub)

	//test address
	addr, err := PubToAddress(pub)
	assert.Nil(t, err)
	assert.Equal(t, addr, secp256k1Addr)
	naddr, err := wallet.NewAddress(0)
	assert.Nil(t, err)
	assert.Equal(t, addr, naddr)

	tpub, err := PrivkeyToPub(TypeYcc, types.SECP256K1, priv)
	assert.Nil(t, err)
	assert.Equal(t, tpub, pub)
}

func TestSm2PrivPub(t *testing.T) {
	wallet, err := NewWalletFromMnemonic(TypeYcc, types.SM2, mnem)
	assert.Nil(t, err)
	priv, pub, err := wallet.NewKeyPair(0)
	assert.Nil(t, err)
	assert.Equal(t, 32, len(priv))
	assert.Equal(t, sm2Pub, hex.EncodeToString(pub))

	//test address
	addr, err := PubToAddress(pub)
	assert.Nil(t, err)
	assert.Equal(t, sm2Addr, addr)
	naddr, err := wallet.NewAddress(0)
	assert.Nil(t, err)
	assert.Equal(t, addr, naddr)

	tpub, err := PrivkeyToPub(TypeYcc, types.SM2, priv)
	assert.Nil(t, err)
	assert.Equal(t, tpub, pub)
}
