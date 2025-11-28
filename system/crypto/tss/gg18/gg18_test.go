package gg18_test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
	"testing"
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
