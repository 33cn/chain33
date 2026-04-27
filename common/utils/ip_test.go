package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPublicIP(t *testing.T) {
	assert.False(t, IsPublicIP(""))
	assert.False(t, IsPublicIP("invalid"))
	assert.False(t, IsPublicIP("127.0.0.1"))
	assert.False(t, IsPublicIP("::1"))
	assert.False(t, IsPublicIP("10.0.0.1"))
	assert.False(t, IsPublicIP("10.255.255.255"))
	assert.False(t, IsPublicIP("172.16.0.1"))
	assert.False(t, IsPublicIP("172.31.255.255"))
	assert.False(t, IsPublicIP("192.168.1.1"))
	assert.False(t, IsPublicIP("192.168.0.1"))
	assert.True(t, IsPublicIP("8.8.8.8"))
	assert.True(t, IsPublicIP("1.1.1.1"))
	assert.True(t, IsPublicIP("172.32.0.1"))
	assert.True(t, IsPublicIP("11.0.0.1"))
	assert.True(t, IsPublicIP("193.169.1.1"))
	assert.False(t, IsPublicIP("224.0.0.1"))
}

func TestLocalIPv4s(t *testing.T) {
	ips, err := LocalIPv4s()
	assert.Nil(t, err)
	assert.NotNil(t, ips)
}
