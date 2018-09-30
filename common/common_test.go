package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandString(t *testing.T) {
	str := GetRandString(10)
	assert.Len(t, str, 10)
}

func TestRandStringLen(t *testing.T) {
	str := GetRandBytes(10, 20)
	if len(str) < 10 || len(str) > 20 {
		t.Error("rand str len")
	}
}
