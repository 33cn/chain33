// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package difficulty

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBigToCompactNegative(t *testing.T) {
	neg := big.NewInt(-1)
	c := BigToCompact(neg)
	assert.NotEqual(t, uint32(0), c)
}

func TestBigToCompactZero(t *testing.T) {
	c := BigToCompact(big.NewInt(0))
	assert.Equal(t, uint32(0), c)
}

func TestBigToCompactSmall(t *testing.T) {
	c := BigToCompact(big.NewInt(1))
	assert.Greater(t, c, uint32(0))
}

func TestBigToCompactLarge(t *testing.T) {
	n := new(big.Int).Lsh(big.NewInt(1), 256)
	c := BigToCompact(n)
	assert.Greater(t, c, uint32(0))
}
