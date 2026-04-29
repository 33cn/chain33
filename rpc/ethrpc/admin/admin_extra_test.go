package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatadir(t *testing.T) {
	result, err := admin.Datadir()
	assert.Nil(t, err)
	assert.NotEmpty(t, result)
}
