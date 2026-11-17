package types

import (
	"bytes"
	"math/big"
	"testing"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestMakeDERSigToRSVRejectsOversizedR(t *testing.T) {
	oversized := bytes.Repeat([]byte{0x80}, 33)
	der := []byte{0x30, byte(4 + len(oversized) + 3), 0x02, byte(len(oversized))}
	der = append(der, oversized...)
	der = append(der, 0x02, 0x01, 0x01)

	signer := etypes.NewEIP155Signer(big.NewInt(1))
	_, _, _, err := makeDERSigToRSV(signer, der)
	require.Error(t, err)
}

func TestMakeDERSigToRSVRejectsOversizedS(t *testing.T) {
	oversized := bytes.Repeat([]byte{0x80}, 33)
	der := []byte{0x30, byte(4 + 1 + 2 + len(oversized)), 0x02, 0x01, 0x01, 0x02, byte(len(oversized))}
	der = append(der, oversized...)

	signer := etypes.NewEIP155Signer(big.NewInt(1))
	_, _, _, err := makeDERSigToRSV(signer, der)
	require.Error(t, err)
}

// A valid DER-encoded S with the high bit set has a leading 0x00 and is 33
// bytes long. It must be accepted after the leading zero is stripped, not
// rejected by the 32-byte bound.
func TestMakeDERSigToRSVAcceptsLeadingZeroS(t *testing.T) {
	rBytes := bytes.Repeat([]byte{0x01}, 32)
	sBytes := bytes.Repeat([]byte{0x02}, 32)
	sBytes[0] = 0x80
	der := makeDERsignature(rBytes, sBytes)

	signer := etypes.NewEIP155Signer(big.NewInt(1))
	r, s, _, err := makeDERSigToRSV(signer, der)
	require.NoError(t, err)
	require.Equal(t, new(big.Int).SetBytes(rBytes), r)
	require.Equal(t, new(big.Int).SetBytes(sBytes), s)
}
