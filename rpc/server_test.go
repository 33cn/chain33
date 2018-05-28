package rpc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckWhitlist(t *testing.T) {
	address := "0.0.0.0"
	assert.False(t, checkWhitlist(address))

	address = "192.168.3.1"
	whitlist[address] = true
	assert.False(t, checkWhitlist("192.168.3.2"))

	whitlist["0.0.0.0"] = true
	assert.True(t, checkWhitlist(address))
	assert.True(t, checkWhitlist("192.168.3.2"))

}
