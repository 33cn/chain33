package dapp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRegisterKVExpiredCheckerFull(t *testing.T) {
	RegisterKVExpiredChecker("test_checker", func(key, value []byte) bool {
		return len(key) > 0
	})

	f, ok := LoadKVExpiredChecker("test_checker")
	assert.True(t, ok)
	assert.NotNil(t, f)
	assert.True(t, f([]byte("k"), []byte("v")))

	list := KVExpiredCheckerList()
	assert.Contains(t, list, "test_checker")
}
