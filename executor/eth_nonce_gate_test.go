package executor

import (
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/client/mocks"
	"github.com/33cn/chain33/common/crypto"
	erpctypes "github.com/33cn/chain33/rpc/ethrpc/types"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	ecommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

// These tests exercise the pre-fee nonce gate in execTx/execTxGroup against a seeded
// LocalDB nonce record, so no EVM executor is needed: the gate returns before execFee.
// The key and value formats are the ones the EVM executor writes and reads, and the
// seed goes straight into the LocalDB cache because a cache hit keeps the tests free of
// a running blockchain module (LocalDB would otherwise forward reads over the queue).

const (
	testEthSignKeyHex = "7939624566468cfa3cb2c9f39d5ad83bdc7cf4356bfd1a7b8094abda6b0699d1"
	testEthChainID    = 3999
)

// newEthSignedCarrier assembles an eth-signed chain33 carrier exactly like the JSON-RPC
// path does, so the envelope and signature checks see a canonical transaction.
func newEthSignedCarrier(t *testing.T, cfg *types.Chain33Config, to ecommon.Address, nonce uint64, data []byte) *types.Transaction {
	t.Helper()

	sk, err := ethcrypto.ToECDSA(ecommon.FromHex(testEthSignKeyHex))
	require.NoError(t, err)
	etx := ethtypes.NewTransaction(nonce, to, big.NewInt(0), 3000000, big.NewInt(10e9), data)
	signer := ethtypes.NewEIP155Signer(big.NewInt(testEthChainID))
	signed, err := ethtypes.SignTx(etx, signer, sk)
	require.NoError(t, err)

	v, r, s := signed.RawSignatureValues()
	cv, err := erpctypes.CaculateRealV(v, signed.ChainId().Uint64(), signed.Type())
	require.NoError(t, err)
	sig := make([]byte, 65)
	copy(sig[32-len(r.Bytes()):32], r.Bytes())
	copy(sig[64-len(s.Bytes()):64], s.Bytes())
	sig[64] = cv

	crypto.Init(cfg.GetModuleConfig().Crypto, map[string][]byte{
		secp256k1eth.Name: cfg.GetSubConfig().Crypto[secp256k1eth.Name],
	})
	pubkey, err := ethcrypto.Ecrecover(signer.Hash(signed).Bytes(), sig)
	require.NoError(t, err)

	carrier := erpctypes.AssembleChain33Tx(signed, sig, pubkey, cfg)
	require.NotNil(t, carrier)
	return carrier
}

// newNonceLocalDB returns a LocalDB holding the EVM nonce record for addr. The constructor
// is the production one; only the queue side is mocked, and the record is seeded into the
// cache so reads never leave the process.
func newNonceLocalDB(t *testing.T, addr string, nonce int64) *LocalDB {
	t.Helper()

	api := mocks.NewQueueProtocolAPI(t)
	api.On("LocalNew", false).Return(&types.Int64{Data: 1}, nil)
	ldb := NewLocalDB(nil, api, false).(*LocalDB)
	ldb.cache.Set(
		secp256k1eth.CaculCoinsEvmAccountKey(addr),
		types.Encode(&types.EvmAccountNonce{Addr: addr, Nonce: nonce}),
	)
	return ldb
}

func testEthExecCtx() *executorCtx {
	return &executorCtx{
		height:     1,
		blocktime:  time.Now().Unix(),
		difficulty: 1,
	}
}

// TestExecTxEthNonceGateRejectsStaleCarrier checks the execTx call site: a carrier whose
// embedded eth nonce does not match the LocalDB record is rejected before execFee, so the
// signer is not charged. The receipt is nil, which is the pre-fee path.
func TestExecTxEthNonceGateRejectsStaleCarrier(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	// Keep the fee path out of the picture: the assertion is about reaching (or not
	// reaching) execFee, not about balances, and a zero rate keeps the mutated run from
	// querying the store module.
	cfg.SetTxFeeConfig(0, 1e7, 1e9)
	cfg.SetDappFork("evm", "ForkEvmExecNonceV2", 0)
	types.AllowUserExec = append(types.AllowUserExec, []byte("evm"))

	carrier := newEthSignedCarrier(t, cfg, ecommon.HexToAddress("0x1111111111111111111111111111111111111111"), 0, nil)
	ldb := newNonceLocalDB(t, carrier.From(), 5)

	execute := newExecutor(testEthExecCtx(), exec, ldb, nil, nil)
	receipt, err := execute.execTx(nil, carrier, 0)

	require.Error(t, err)
	require.Nil(t, receipt, "the gate must run before execFee, so no receipt exists")
	require.Contains(t, strings.ToLower(err.Error()), "nonce")
}

// TestExecTxGroupEthNonceGateRejectsStaleHead checks the group call site: the group head
// pays the fee for every member, so it gets the same pre-fee gate.
func TestExecTxGroupEthNonceGateRejectsStaleHead(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	cfg.SetTxFeeConfig(0, 1e7, 1e9)
	cfg.SetDappFork("evm", "ForkEvmExecNonceV2", 0)
	// The envelope rule requires the single-transaction shape, so a carrier cannot be a
	// group member while it is active. It is a different fork from the V2 nonce gate;
	// disabling it here isolates the group call site without weakening that gate.
	cfg.SetFork(types.ForkEthSignEnvelope, testEthExecCtx().height+100)

	carrier := newEthSignedCarrier(t, cfg, ecommon.HexToAddress("0x2222222222222222222222222222222222222222"), 0, nil)
	_, priv := util.Genaddress()
	member := util.CreateNoneTx(cfg, priv)
	group, err := types.CreateTxGroup([]*types.Transaction{carrier, member}, cfg.GetMinTxFeeRate())
	require.NoError(t, err)
	require.Equal(t, int32(2), group.Txs[0].GroupCount)

	ldb := newNonceLocalDB(t, carrier.From(), 5)
	execute := newExecutor(testEthExecCtx(), exec, ldb, group.GetTxs(), nil)
	receipts, err := execute.execTxGroup(group.GetTxs(), 0)

	require.Error(t, err)
	require.Nil(t, receipts, "the gate must run before execFee, so no receipt exists")
	require.Contains(t, strings.ToLower(err.Error()), "nonce")
}

// TestExecTxEthNonceGateSkipsProxyExec checks that the gate stays out of the proxy branch.
// There the transaction handed to the fee step is the inner chain33 transaction, whose
// Nonce is payload metadata and not the EVM nonce; the carrier's own nonce was already
// checked. A gate applied to the inner transaction would fail with "nonce too high"
// because its nonce (7) is ahead of the recorded one (0).
func TestExecTxEthNonceGateSkipsProxyExec(t *testing.T) {
	exec, _ := initEnv(types.GetDefaultCfgstring())
	cfg := exec.client.GetConfig()
	// With a zero rate execFee returns without touching accounts, and an execution error
	// is recorded in the receipt instead of being returned. That lets this test run with
	// no store module while still exercising the proxy branch.
	cfg.SetTxFeeConfig(0, 1e7, 1e9)
	cfg.SetDappFork("evm", "ForkEvmExecNonceV2", 0)

	proxyAddrHex := cfg.GetModuleConfig().Exec.ProxyExecAddress
	require.NotEmpty(t, proxyAddrHex)
	to, _ := util.Genaddress()
	inner := util.CreateCoinsTx(cfg, nil, to, 1000)
	// The inner nonce is deliberately unrelated to the recorded EVM nonce.
	inner.Nonce = 7
	// A payload the coins driver cannot decode makes the later execution stage fail
	// without touching state, after the gate has had its chance to run.
	inner.Payload = []byte("not a valid coins action")

	carrier := newEthSignedCarrier(t, cfg, ecommon.HexToAddress(proxyAddrHex), 0, types.Encode(inner))
	ldb := newNonceLocalDB(t, carrier.From(), 0)

	execute := newExecutor(testEthExecCtx(), exec, ldb, nil, nil)
	_, err := execute.execTx(nil, carrier, 0)

	// The invalid payload fails in the execution stage, whose error is folded into the
	// receipt instead of being returned. If the gate had run on the inner transaction,
	// execTx would instead return here with "nonce too high" for its nonce 7.
	require.NoError(t, err)
}
