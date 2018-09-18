package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTypeReflact(t *testing.T) {
	ty := NewType()
	assert.NotNil(t, ty)
}
