package gg18_test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

func TestGG18(t *testing.T) {

	pk, pub, err := crypto.GenerateECDSAKeyPair(rand.Reader)
	require.NoError(t, err)
	pid, err := peer.IDFromPublicKey(pub)
	require.NoError(t, err)
	pkBytes, err := crypto.MarshalPrivateKey(pk)
	fmt.Println(base64.StdEncoding.EncodeToString(pkBytes))
	fmt.Println(pid.String())
}

func TestConvertPubkey(t *testing.T) {

	priv, err := btcec.NewPrivateKey()
	require.NoError(t, err)
	priv.PubKey()
	pub1 := priv.ToECDSA().PublicKey
	var x, y btcec.FieldVal
	x.SetByteSlice(pub1.X.Bytes())
	res := y.SetByteSlice(pub1.Y.Bytes())
	println(res)
	pub := btcec.NewPublicKey(&x, &y)
	require.True(t, pub.IsEqual(priv.PubKey()))
}
