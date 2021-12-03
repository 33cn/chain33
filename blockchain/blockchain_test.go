// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalcHeightToBlockHeaderKey(t *testing.T) {
	key := calcHeightToBlockHeaderKey(1)
	require.Equal(t, key, []byte("HH:000000000001"))
	key = calcHeightToBlockHeaderKey(0)
	require.Equal(t, key, []byte("HH:000000000000"))
	key = calcHeightToBlockHeaderKey(10)
	require.Equal(t, key, []byte("HH:000000000010"))
}
