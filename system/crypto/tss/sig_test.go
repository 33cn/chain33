// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tss

import (
	"math/big"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2/ecdsa"
	"github.com/stretchr/testify/require"
)

// TestBuildBtcecSignatureNilComponents covers the exported-API panic surface: a nil R or S used to
// reach big.Int.Bytes and panic. Both must return an error instead, including when both are nil.
func TestBuildBtcecSignatureNilComponents(t *testing.T) {
	cases := []struct {
		name string
		r, s *big.Int
	}{
		{"nil R", nil, big.NewInt(1)},
		{"nil S", big.NewInt(1), nil},
		{"both nil", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var (
				sig *ecdsa.Signature
				err error
			)
			require.NotPanics(t, func() { sig, err = BuildBtcecSignature(tc.r, tc.s) })
			require.Error(t, err)
			require.Nil(t, sig)
		})
	}

	// A complete pair still converts.
	sig, err := BuildBtcecSignature(big.NewInt(1), big.NewInt(2))
	require.NoError(t, err)
	require.NotNil(t, sig)
}
