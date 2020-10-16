package net

import (
	"testing"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelayAddrs_OnlyFactory(t *testing.T) {
	relay := "/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi"
	f := WithRelayAddrs([]string{relay})

	a, err := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/33201/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG")
	require.NoError(t, err)
	t.Log("addrstring", a.String())
	addrs := []ma.Multiaddr{a}

	result := f(addrs)
	t.Log("result", len(result), "addrs", result)
	assert.Equal(t, 2, len(result), "Unexpected number of addresses")

	expected := "/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi/p2p-circuit/ip4/127.0.0.1/tcp/33201/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG"
	assert.Equal(t, expected, result[1].String(), "Address at index 1 (%s) is not the expected p2p-circuit address", result[1].String())
}

func TestRelayAddrs_UseNonRelayAddrs(t *testing.T) {
	relay := "/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi"
	f := WithRelayAddrs([]string{relay})

	expected := []string{
		"/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi/p2p-circuit/ip4/127.0.0.1/tcp/33201/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG",
		"/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi/p2p-circuit/ip4/127.0.0.1/tcp/33203/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG",
	}

	addrs := make([]ma.Multiaddr, len(expected))
	for i, addr := range expected {
		a, err := ma.NewMultiaddr(addr)
		require.NoError(t, err)
		addrs[i] = a
	}

	result := f(addrs)
	t.Log("result", result)
	assert.Equal(t, 2, len(result))

}

func Test_WithRelayAddrs(t *testing.T) {
	addrF := WithRelayAddrs([]string{"/ip4/127.0.0.1/tcp/6660/p2p/QmQ7zhY7nGY66yK1n8hLGevfVyjbtvHSgtZuXkCH9oTrgi"})
	assert.NotNil(t, addrF)
	var testAddr = "/ip4/127.0.0.1/tcp/33201/p2p/QmaXZhW44pwQxBSeLkE5FNeLz8tGTTEsRciFg1DNWXXrWG"
	a, err := ma.NewMultiaddr(testAddr)
	require.NoError(t, err)
	maddrs := addrF([]ma.Multiaddr{a})
	assert.Equal(t, len(maddrs), 2)

	addrF = WithRelayAddrs([]string{})
	maddrs = addrF([]ma.Multiaddr{a})
	assert.Equal(t, len(maddrs), 1)

}
