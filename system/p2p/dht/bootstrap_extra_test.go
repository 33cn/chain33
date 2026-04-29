package dht

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenAddrInfo(t *testing.T) {
	// Valid multiaddr
	info := genAddrInfo("/ip4/127.0.0.1/tcp/12345/p2p/16Uiu2HAmK9PAPYoTzHnobzB5nQFnY7p9ZVcJYQ1BgzKCr7izAhbJ")
	assert.NotNil(t, info)

	// Invalid multiaddr
	info2 := genAddrInfo("not a valid multiaddr")
	assert.Nil(t, info2)
}

func TestGenAddrInfos(t *testing.T) {
	// Empty list
	assert.Nil(t, genAddrInfos(nil))
	assert.Nil(t, genAddrInfos([]string{}))

	// Valid list
	infos := genAddrInfos([]string{
		"/ip4/127.0.0.1/tcp/12345/p2p/16Uiu2HAmK9PAPYoTzHnobzB5nQFnY7p9ZVcJYQ1BgzKCr7izAhbJ",
		"invalid",
	})
	assert.Equal(t, 1, len(infos))
}
