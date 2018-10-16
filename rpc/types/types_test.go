package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeUserWrite(t *testing.T) {
	payload := []byte("#md#hello#world")
	data := decodeUserWrite(payload)
	assert.Equal(t, data, &UserWrite{Topic: "md", Content: "hello#world"})

	payload = []byte("hello#world")
	data = decodeUserWrite(payload)
	assert.Equal(t, data, &UserWrite{Topic: "", Content: "hello#world"})

	payload = []byte("123#hello#wzw")
	data = decodeUserWrite(payload)
	assert.NotEqual(t, data, &UserWrite{Topic: "123", Content: "hello#world"})
}
