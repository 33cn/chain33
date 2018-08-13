package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainConfig(t *testing.T) {
	SetChainConfig("a", true)
	_, err := GetChainConfig("b")
	assert.Equal(t, err, ErrNotFound)

	adata, err := GetChainConfig("a")
	assert.Equal(t, err, nil)
	assert.Equal(t, adata.(bool), true)
}
