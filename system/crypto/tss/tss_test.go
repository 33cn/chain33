package tss

import (
	"crypto/ecdsa"
	"fmt"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/require"
)

func TestConvertDKGResult(t *testing.T) {

	priv, err := btcec.NewPrivateKey()
	require.NoError(t, err)
	pub := priv.PubKey()
	dkgRes := &DKGResult{
		PubX: pub.X().Bytes(),
		PubY: pub.Y().Bytes(),
		Bks:  make(map[string]*BK, 1),
	}
	dkgRes.Bks["test"] = &BK{X: pub.X().Bytes()}

	d, err := ConvertDKGResult([]string{"test"}, dkgRes)
	require.NoError(t, err)
	pub1 := &ecdsa.PublicKey{
		X: d.PublicKey.GetX(),
		Y: d.PublicKey.GetY(),
	}

	require.Equal(t, pub1.X.String(), pub.X().String())
	require.Equal(t, pub1.Y.String(), pub.Y().String())
	d1 := NewDKGResult(d)
	require.Equal(t, dkgRes.PubX, d1.PubX)
}

func TestRegisterMsgHandler_DuplicatePanics(t *testing.T) {
	// Use a unique name so we don't conflict with gg18 init or other tests.
	name := "test_dup_" + fmt.Sprintf("%d", time.Now().UnixNano())
	var recovered interface{}
	func() {
		defer func() { recovered = recover() }()
		RegisterMsgHandler(name, func(*MessageWrapper) {})
		RegisterMsgHandler(name, func(*MessageWrapper) {}) // should panic
	}()
	require.NotNil(t, recovered, "expected panic on duplicate RegisterMsgHandler")
	require.Contains(t, fmt.Sprint(recovered), "duplicate handler")
	require.Contains(t, fmt.Sprint(recovered), name)
}
