package mempool

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
	"github.com/33cn/chain33/system/address/eth"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	stypes "github.com/33cn/chain33/system/crypto/secp256k1eth/types"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"

	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
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

// initEmbeddedEthEnv prepares the process-global state the secp256k1eth verifier reads.
// crypto.Init is not called anywhere in this package: without it coinsPrecision is 0 and
// verifying an embedded transaction divides by zero.
func initEmbeddedEthEnv(t *testing.T) (*ecdsa.PrivateKey, crypto.PubKey) {
	t.Helper()
	oldChainID := secp256k1eth.GetEvmChainID()
	crypto.Init(&crypto.Config{}, map[string][]byte{
		secp256k1eth.Name: []byte(`{"evmChainID": 0, "coinsPrecision": 100000000}`),
	})
	t.Cleanup(func() {
		crypto.Init(&crypto.Config{}, map[string][]byte{
			secp256k1eth.Name: []byte(fmt.Sprintf(`{"evmChainID": %d, "coinsPrecision": 100000000}`, oldChainID)),
		})
	})

	driver := secp256k1eth.Driver{}
	key, err := common.FromHex(hexPirv)
	require.Nil(t, err)
	priv, err := ethcrypto.ToECDSA(key)
	require.Nil(t, err)
	pub, err := driver.PubKeyFromBytes(ethcrypto.FromECDSAPub(&priv.PublicKey))
	require.Nil(t, err)
	return priv, pub
}

// TestCheckTxGroupEmbeddedEthNonce repeats the group nonce checks with members that are
// eth-signature carriers: the chain33 transaction wraps a signed Ethereum transaction in
// Note, which is the shape a re-wrapped transaction can be built from.
func TestCheckTxGroupEmbeddedEthNonce(t *testing.T) {
	priv, pub := initEmbeddedEthEnv(t)
	cfg := types.NewChain33Config(types.MergeCfg(
		types.ReadFile("../../cmd/chain33/chain33.test.toml"),
		types.ReadFile("../../cmd/chain33/chain33.fork.toml")))
	feeRate := cfg.GetMinTxFeeRate()
	execAddr := address.ExecAddress("evm")

	newMember := func(nonce int64, to ecommon.Address) *types.Transaction {
		t.Helper()
		etx := etypes.NewTx(&etypes.DynamicFeeTx{
			ChainID:   big.NewInt(0), // secp256k1eth falls back to 0 when crypto.Init has not run
			Nonce:     uint64(nonce),
			GasTipCap: big.NewInt(1),
			GasFeeCap: big.NewInt(1),
			Gas:       1000000,
			To:        &to,
			Value:     big.NewInt(0),
		})
		raw, err := etx.MarshalBinary()
		require.Nil(t, err)
		sig, err := ethcrypto.Sign(etypes.NewLondonSigner(etx.ChainId()).Hash(etx).Bytes(), priv)
		require.Nil(t, err)

		tx := &types.Transaction{
			Execer:  []byte("evm"),
			Payload: types.Encode(&stypes.EVMAction4Chain33{Amount: 0, Para: to.Bytes(), Note: ecommon.Bytes2Hex(raw), ContractAddr: execAddr}),
			To:      execAddr,
			Nonce:   nonce,
			ChainID: cfg.GetChainID(),
			Signature: &types.Signature{
				Ty:        types.EncodeSignID(secp256k1eth.ID, eth.ID),
				Pubkey:    pub.Bytes(),
				Signature: sig,
			},
		}
		tx.SetExpire(cfg, 3*time.Hour)
		return tx
	}
	newGroup := func(t *testing.T, members ...*types.Transaction) *types.Transaction {
		t.Helper()
		group, err := types.CreateTxGroup(members, feeRate)
		require.Nil(t, err)
		return group.Tx()
	}
	addrA := ecommon.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	addrB := ecommon.HexToAddress("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")

	run := func(t *testing.T, tx *types.Transaction) *types.Reply {
		t.Helper()
		_, mem := initEnv(10)
		defer mem.Close()
		msg := mem.client.NewMessage("mempool", types.EventTx, tx)
		mem.client.Send(msg, true)
		msg, err := mem.client.Wait(msg)
		require.Nil(t, err)
		reply := msg.GetData().(*types.Reply)
		if reply.GetIsOk() {
			assert.Equal(t, 1, mem.Size())
		} else {
			assert.Equal(t, 0, mem.Size())
		}
		return reply
	}

	t.Run("valid contiguous", func(t *testing.T) {
		reply := run(t, newGroup(t, newMember(1, addrA), newMember(2, addrB)))
		assert.True(t, reply.GetIsOk(), string(reply.GetMsg()))
	})
	t.Run("duplicate nonce", func(t *testing.T) {
		reply := run(t, newGroup(t, newMember(1, addrA), newMember(1, addrB)))
		assert.False(t, reply.GetIsOk())
		assert.Equal(t, "non-contiguous eth nonce in tx group", string(reply.GetMsg()))
	})
	t.Run("non contiguous", func(t *testing.T) {
		reply := run(t, newGroup(t, newMember(1, addrA), newMember(3, addrB)))
		assert.False(t, reply.GetIsOk())
		assert.Equal(t, "non-contiguous eth nonce in tx group", string(reply.GetMsg()))
	})
	t.Run("stale non-first member", func(t *testing.T) {
		reply := run(t, newGroup(t, newMember(1, addrA), newMember(0, addrB)))
		assert.False(t, reply.GetIsOk())
		assert.Equal(t, types.ErrLowNonce.Error(), string(reply.GetMsg()))
	})
}
