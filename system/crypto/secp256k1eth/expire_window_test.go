package secp256k1eth_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	stypes "github.com/33cn/chain33/system/crypto/secp256k1eth/types"
	ctypes "github.com/33cn/chain33/types"
)

// TestEthExpireVariantsAccepted pins the shape the fee rule has to close: the envelope rule
// bounds Expire to a window instead of pinning it, so one embedded Ethereum transaction
// yields several chain33 transactions that all pass the rule and all carry a distinct hash.
// The duplicate check keys on that hash, so each variant reaches execution on its own; the
// pre-fee nonce gate in the executor is what keeps them from being charged.
func TestEthExpireVariantsAccepted(t *testing.T) {
	cfg, _, _ := newTestEnv(t, false)
	base := newEthTx(7, &testRecipient, nil)
	tx := assembleWith(t, cfg, base)
	require.NoError(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))

	hashes := map[string]struct{}{string(asChain33Tx(t, tx).Hash()): {}}
	for _, delta := range []int64{-1, -60, -3600, -10800} {
		variant := proto.Clone(tx).(*stypes.TransactionChain33)
		variant.Expire += delta
		require.NoError(t, checkEnvelope(t, cfg, variant, envelopeForkHeight),
			"expire %d is inside the window", variant.Expire)
		hash := string(asChain33Tx(t, variant).Hash())
		require.NotContains(t, hashes, hash, "variant must not collide with the assembled transaction")
		hashes[hash] = struct{}{}
	}
}

// TestEthExpireIgnoresBlockTimeBeforeV2 pins the replay guarantee: before ForkEthSignEnvelopeV2
// the window is measured against the local clock, so an older block replays with the verdict
// it got when it was mined. A block time four hours in the past would reject the envelope if
// it were used, which is exactly what must not happen before the fork.
func TestEthExpireIgnoresBlockTimeBeforeV2(t *testing.T) {
	cfg, _, _ := newTestEnv(t, false)
	// The local test config activates every fork; schedule V2 after the height under test.
	cfg.SetFork(ctypes.ForkEthSignEnvelopeV2, envelopeForkHeight+100)
	base := newEthTx(7, &testRecipient, nil)
	tx := assembleWith(t, cfg, base)
	oldBlockTime := time.Now().Unix() - int64(4*time.Hour/time.Second)

	require.NoError(t, asChain33Tx(t, tx).CheckAtBlockTime(cfg, envelopeForkHeight, 0, 0, oldBlockTime))

	// The same call after the fork uses the block time and rejects it.
	cfg.SetFork(ctypes.ForkEthSignEnvelopeV2, envelopeForkHeight)
	require.Error(t, asChain33Tx(t, tx).CheckAtBlockTime(cfg, envelopeForkHeight, 0, 0, oldBlockTime))
}

// TestEthExpireUsesBlockTime requires the window to be evaluated against the block time the
// caller supplies, not against the verifying node's clock: whether a block is valid may
// depend on its contents only.
func TestEthExpireUsesBlockTime(t *testing.T) {
	cfg, _, _ := newTestEnv(t, false)
	cfg.SetFork(ctypes.ForkEthSignEnvelopeV2, envelopeForkHeight)
	base := newEthTx(7, &testRecipient, nil)
	tx := assembleWith(t, cfg, base)

	blockTime := time.Now().Unix()
	late := proto.Clone(tx).(*stypes.TransactionChain33)
	late.Expire = blockTime + int64(4*time.Hour/time.Second)

	// Four hours ahead of the block is outside the three hour window.
	require.Error(t, asChain33Tx(t, late).CheckAtBlockTime(cfg, envelopeForkHeight, 0, 0, blockTime))
	// The same transaction is inside the window of a block one and a half hours later,
	// which is what makes the verdict a function of the block alone.
	require.NoError(t, asChain33Tx(t, late).CheckAtBlockTime(cfg, envelopeForkHeight, 0, 0,
		blockTime+int64(90*time.Minute/time.Second)))
}
