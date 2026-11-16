package executor

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/types"
)

// TestCompareEthNonce pins the rule that keeps a block producer from charging the signer of
// an eth-channel transaction once per rewritten Expire: the nonce is compared before the fee
// is taken, and a missing record counts as nonce zero, exactly like the EVM executor.
func TestCompareEthNonce(t *testing.T) {
	record := func(nonce int64) []byte {
		return types.Encode(&types.EvmAccountNonce{Addr: "0xabc", Nonce: nonce})
	}
	missing := errors.New("not found")

	require.NoError(t, compareEthNonce(nil, missing, 0))
	require.Error(t, compareEthNonce(nil, missing, 7))

	require.NoError(t, compareEthNonce(record(7), nil, 7))
	require.Equal(t, types.ErrLowNonce, compareEthNonce(record(7), nil, 6))
	// A record that cannot be decoded behaves like nonce zero, not like a panic.
	require.Error(t, compareEthNonce([]byte("garbage"), nil, 7))
}
