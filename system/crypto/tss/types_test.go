package tss

import (
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/stretchr/testify/require"
)

func TestComposeProtocol(t *testing.T) {
	require.Equal(t, "/gg18/dkg", ComposeProtocol("/gg18/dkg", ""))
	require.Equal(t, "/gg18/dkg|sess1", ComposeProtocol("/gg18/dkg", "sess1"))
}

func TestSplitProtocol(t *testing.T) {
	proto, session := SplitProtocol("")
	require.Equal(t, "", proto)
	require.Equal(t, "", session)

	proto, session = SplitProtocol("/gg18/dkg")
	require.Equal(t, "/gg18/dkg", proto)
	require.Equal(t, "", session)

	proto, session = SplitProtocol("/gg18/dkg|sess1")
	require.Equal(t, "/gg18/dkg", proto)
	require.Equal(t, "sess1", session)

	// separator in middle
	proto, session = SplitProtocol("/gg18/dkg|sess1|extra")
	require.Equal(t, "/gg18/dkg|sess1", proto)
	require.Equal(t, "extra", session)
}

func TestParseBtcecPublicKey(t *testing.T) {
	priv, err := btcec.NewPrivateKey()
	require.NoError(t, err)
	pub := priv.PubKey()

	result := &DKGResult{
		PubX: pub.X().Bytes(),
		PubY: pub.Y().Bytes(),
	}
	parsed, err := ParseBtcecPublicKey(result)
	require.NoError(t, err)
	require.True(t, pub.IsEqual(parsed))
}

func TestParseBtcecPublicKey_InvalidX(t *testing.T) {
	// X too large for field (33 bytes with high bits set can cause SetByteSlice to return true for overflow)
	invalid := &DKGResult{
		PubX: make([]byte, 33),
		PubY: make([]byte, 32),
	}
	for i := range invalid.PubX {
		invalid.PubX[i] = 0xff
	}
	_, err := ParseBtcecPublicKey(invalid)
	require.Error(t, err)
}
