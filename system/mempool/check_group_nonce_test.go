package mempool

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/crypto"
	erpctypes "github.com/33cn/chain33/rpc/ethrpc/types"
	"github.com/33cn/chain33/system/address/eth"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
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

	// Ordinary chain33 transactions signed with an eth key carry a random nonce
	// (wallet/sendtx.go), so a group from a single sender does not have contiguous
	// nonces. Only members that embed an Ethereum transaction may be required to.
	for _, pair := range [][2]int64{
		{7378697629483820646, 3614888832304827231},
		{12345, 67890},
	} {
		t.Run(fmt.Sprintf("random nonces %d/%d", pair[0], pair[1]), func(t *testing.T) {
			_, mem := initEnv(10)
			defer mem.Close()

			msg := mem.client.NewMessage("mempool", types.EventTx, newEthGroup(t, pair[0], pair[1]))
			mem.client.Send(msg, true)
			msg, err := mem.client.Wait(msg)
			require.Nil(t, err)
			reply := msg.GetData().(*types.Reply)
			assert.True(t, reply.GetIsOk(), string(reply.GetMsg()))
			assert.Equal(t, 1, mem.Size())
		})
	}
}

// initEmbeddedEthEnv prepares the process-global state the secp256k1eth verifier reads,
// exactly as the node does: the chain parameters are copied into the driver sub config.
func initEmbeddedEthEnv(t *testing.T, cfg *types.Chain33Config) (*ecdsa.PrivateKey, crypto.PubKey) {
	t.Helper()
	oldChainID := secp256k1eth.GetEvmChainID()
	crypto.Init(&crypto.Config{}, map[string][]byte{
		secp256k1eth.Name: cfg.GetSubConfig().Crypto[secp256k1eth.Name],
	})
	t.Cleanup(func() {
		crypto.Init(&crypto.Config{}, map[string][]byte{
			secp256k1eth.Name: []byte(fmt.Sprintf(`{"evmChainID": %d}`, oldChainID)),
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

// TestCheckEthSignedEnvelopeFork pins the mempool decision: below the fork the envelope
// rule is off, so an already-mined assembler shape enters the pool; at the fork the same
// shape is rejected because it is not what the current assembler emits.
func TestCheckEthSignedEnvelopeFork(t *testing.T) {
	cfg := types.NewChain33Config(types.MergeCfg(
		types.ReadFile("../../cmd/chain33/chain33.test.toml"),
		types.ReadFile("../../cmd/chain33/chain33.fork.toml")))
	priv, pub := initEmbeddedEthEnv(t, cfg)

	to := ecommon.HexToAddress("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	etx := etypes.NewTx(&etypes.DynamicFeeTx{
		ChainID:   big.NewInt(secp256k1eth.GetEvmChainID()),
		Nonce:     1,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       1000000,
		To:        &to,
		Value:     big.NewInt(0),
	})
	sig, err := ethcrypto.Sign(etypes.NewLondonSigner(etx.ChainId()).Hash(etx).Bytes(), priv)
	require.Nil(t, err)
	carrier := erpctypes.AssembleChain33Tx(etx, sig, pub.Bytes(), cfg)
	require.NotNil(t, carrier)
	// The v1.67.4 assembler charged max(gas, 1e5); this config uses another fee floor,
	// so the legacy shape is distinguishable from the canonical one.
	carrier.Fee = 1e5
	require.True(t, carrier.CheckSign(0))

	for _, tc := range []struct {
		name   string
		fork   int64
		wantOK bool
	}{
		{"before fork", types.MaxHeight, true},
		{"at fork", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, mem := initEnv(10)
			defer mem.Close()
			mem.client.GetConfig().SetFork(types.ForkEthSignEnvelope, tc.fork)

			msg := mem.checkTxs(mem.client.NewMessage("mempool", types.EventTx, carrier))
			if tc.wantOK {
				require.IsType(t, &types.TransactionCache{}, msg.GetData())
			} else {
				require.Error(t, msg.Err())
			}
		})
	}
}
