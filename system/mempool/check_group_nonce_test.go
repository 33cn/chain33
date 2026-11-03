package mempool

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/system/address/eth"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
)

// TestCheckTxGroupEthNonce checks that the eth nonce check applies to every member of a
// transaction group, not only to the group head.
func TestCheckTxGroupEthNonce(t *testing.T) {
	ethCrypto, err := crypto.Load(types.GetSignName("", types.SECP256K1ETH), -1)
	require.Nil(t, err)
	key, err := common.FromHex(hexPirv)
	require.Nil(t, err)
	ethPriv, err := ethCrypto.PrivKeyFromBytes(key)
	require.Nil(t, err)

	cfg := types.NewChain33Config(types.MergeCfg(
		types.ReadFile("../../cmd/chain33/chain33.test.toml"),
		types.ReadFile("../../cmd/chain33/chain33.fork.toml")))
	feeRate := cfg.GetMinTxFeeRate()

	newMember := func(nonce int64) *types.Transaction {
		action := &cty.CoinsAction{
			Value: &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{Amount: amount}},
			Ty:    cty.CoinsActionTransfer,
		}
		return &types.Transaction{
			ChainID: 0,
			Execer:  []byte("coins"),
			Payload: types.Encode(action),
			Expire:  0,
			To:      toAddr,
			Nonce:   nonce,
		}
	}
	newEthGroup := func(t *testing.T, nonces ...int64) *types.Transaction {
		t.Helper()
		members := make([]*types.Transaction, len(nonces))
		for i, nonce := range nonces {
			members[i] = newMember(nonce)
		}
		group, groupErr := types.CreateTxGroup(members, feeRate)
		require.Nil(t, groupErr)
		for _, member := range members {
			member.Sign(types.EncodeSignID(secp256k1eth.ID, eth.ID), ethPriv)
		}
		// Tx packs the signed group into the head; GetTxGroup on the receiver side
		// decodes it from Header.
		return group.Tx()
	}

	// The stale nonce sits on a member after the first one; the head alone is valid.
	t.Run("stale non-first member", func(t *testing.T) {
		_, mem := initEnv(10)
		defer mem.Close()

		msg := mem.client.NewMessage("mempool", types.EventTx, newEthGroup(t, 1, 0))
		mem.client.Send(msg, true)
		msg, err := mem.client.Wait(msg)
		require.Nil(t, err)
		reply := msg.GetData().(*types.Reply)
		assert.False(t, reply.GetIsOk())
		assert.Equal(t, types.ErrLowNonce.Error(), string(reply.GetMsg()))
		assert.Equal(t, 0, mem.Size())
	})

	// A group whose members all carry valid eth nonces is still accepted.
	t.Run("valid members", func(t *testing.T) {
		_, mem := initEnv(10)
		defer mem.Close()

		msg := mem.client.NewMessage("mempool", types.EventTx, newEthGroup(t, 1, 2))
		mem.client.Send(msg, true)
		msg, err := mem.client.Wait(msg)
		require.Nil(t, err)
		reply := msg.GetData().(*types.Reply)
		assert.True(t, reply.GetIsOk())
		assert.Equal(t, 1, mem.Size())
	})
}
