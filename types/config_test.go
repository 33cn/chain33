package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestChainConfig(t *testing.T) {
	S("a", true)
	_, err := G("b")
	assert.Equal(t, err, ErrNotFound)

	adata, err := G("a")
	assert.Equal(t, err, nil)
	assert.Equal(t, adata.(bool), true)
}
