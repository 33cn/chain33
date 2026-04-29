package gg18

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBigIntToModNScalar(t *testing.T) {
	s, err := bigIntToModNScalar(big.NewInt(1))
	assert.Nil(t, err)
	assert.NotNil(t, s)

	s, err = bigIntToModNScalar(big.NewInt(0))
	assert.Nil(t, err)
	assert.NotNil(t, s)
}

func TestBigIntToModNScalarTooLarge(t *testing.T) {
	n := new(big.Int).Lsh(big.NewInt(1), 300)
	_, err := bigIntToModNScalar(n)
	assert.NotNil(t, err)
}

func TestBigIntToModNScalarNegative(t *testing.T) {
	_, err := bigIntToModNScalar(big.NewInt(-1))
	assert.Nil(t, err)
}

