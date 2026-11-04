package secp256k1eth_test

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	chain33crypto "github.com/33cn/chain33/common/crypto"
	_ "github.com/33cn/chain33/system/address"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	stypes "github.com/33cn/chain33/system/crypto/secp256k1eth/types"
	ctypes "github.com/33cn/chain33/types"

	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const (
	testEvmChainID = 3999
	testEvmExecer  = "evm"
	testParaTitle  = "user.p.para."
	// envelopeForkHeight is the height the envelope rule is enabled at in these tests.
	envelopeForkHeight = 100
)

var (
	testPrivKeyHex = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	testAttacker   = ecommon.HexToAddress("0x1111111111111111111111111111111111111111")
	testRecipient  = ecommon.HexToAddress("0x2222222222222222222222222222222222222222")
	// testChecksumRecipient has hex letters, so its EIP-55 spelling differs from its
	// lower-case spelling and an address-format test can tell the two apart.
	testChecksumRecipient = ecommon.HexToAddress("0xAb5801a7D398351b8bE11C439e05C5B3259aeC9B")
	// testProxyAddr is the configured proxy exec address on bityuan.
	testProxyAddr = ecommon.HexToAddress("0x0000000000000000000000000000000000200005")
)

func newTestConfig(para bool) *ctypes.Chain33Config {
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	if para {
		cfg.SetTitleOnlyForTest(testParaTitle)
	}
	return cfg
}

// startEnv initializes the driver exactly as a node does, from the configured
// [crypto.sub.secp256k1eth] sub config.
func startEnv(t *testing.T, cfg *ctypes.Chain33Config) (*ecdsa.PrivateKey, chain33crypto.PubKey) {
	t.Helper()
	return initDriver(t, cfg.GetSubConfig().Crypto[secp256k1eth.Name])
}

// initDriver starts the eth signature driver from an explicit sub config, which is how a
// node passes it the evm chain id and the chain's coin precision.
func initDriver(t *testing.T, sub []byte) (*ecdsa.PrivateKey, chain33crypto.PubKey) {
	t.Helper()
	oldChainID := secp256k1eth.GetEvmChainID()
	chain33crypto.Init(&chain33crypto.Config{}, map[string][]byte{secp256k1eth.Name: sub})
	t.Cleanup(func() {
		chain33crypto.Init(&chain33crypto.Config{}, map[string][]byte{
			secp256k1eth.Name: []byte(fmt.Sprintf(`{"evmChainID": %d}`, oldChainID)),
		})
	})

	priv := privKeyForTest(t)
	driver := secp256k1eth.Driver{}
	pub, err := driver.PubKeyFromBytes(ethcrypto.FromECDSAPub(&priv.PublicKey))
	require.NoError(t, err)
	return priv, pub
}

func privKeyForTest(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := common.FromHex(testPrivKeyHex)
	require.NoError(t, err)
	priv, err := ethcrypto.ToECDSA(key)
	require.NoError(t, err)
	return priv
}

func newTestEnv(t *testing.T, para bool) (*ctypes.Chain33Config, *ecdsa.PrivateKey, chain33crypto.PubKey) {
	t.Helper()
	cfg := newTestConfig(para)
	priv, pub := startEnv(t, cfg)
	cfg.SetFork(ctypes.ForkEthSignEnvelope, envelopeForkHeight)
	return cfg, priv, pub
}

func newEthTx(nonce uint64, to *ecommon.Address, data []byte) *etypes.Transaction {
	return newEthTxValue(nonce, to, data, big.NewInt(0))
}

func newEthTxValue(nonce uint64, to *ecommon.Address, data []byte, value *big.Int) *etypes.Transaction {
	return etypes.NewTx(&etypes.DynamicFeeTx{
		ChainID:   big.NewInt(testEvmChainID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       100000,
		To:        to,
		Value:     value,
		Data:      data,
	})
}

func signEthTx(t *testing.T, priv *ecdsa.PrivateKey, etx *etypes.Transaction) chain33crypto.Signature {
	t.Helper()
	hash := etypes.NewLondonSigner(etx.ChainId()).Hash(etx)
	sig, err := ethcrypto.Sign(hash.Bytes(), priv)
	require.NoError(t, err)
	driver := secp256k1eth.Driver{}
	sigObj, err := driver.SignatureFromBytes(sig)
	require.NoError(t, err)
	return sigObj
}

// assembleWith is the canonical assembly the verifier rebuilds from.
func assembleWith(t *testing.T, cfg *ctypes.Chain33Config, etx *etypes.Transaction) *stypes.TransactionChain33 {
	t.Helper()
	tx, err := stypes.AssembleChain33Tx(etx, cfg.EthAssembleParams(time.Now()))
	require.NoError(t, err)
	return tx
}

func marshalTx(t *testing.T, tx *stypes.TransactionChain33) []byte {
	t.Helper()
	msg, err := proto.Marshal(tx)
	require.NoError(t, err)
	return msg
}

func decodeAction(t *testing.T, tx *stypes.TransactionChain33) *stypes.EVMAction4Chain33 {
	t.Helper()
	var action stypes.EVMAction4Chain33
	require.NoError(t, proto.Unmarshal(tx.Payload, &action))
	return &action
}

func encodeAction(t *testing.T, tx *stypes.TransactionChain33, action *stypes.EVMAction4Chain33) {
	t.Helper()
	payload, err := proto.Marshal(action)
	require.NoError(t, err)
	tx.Payload = payload
}

// asChain33Tx converts the assembled wire transaction to the root representation and
// marks it as an eth-signed carrier, which is the shape Transaction.check inspects.
func asChain33Tx(t *testing.T, tx *stypes.TransactionChain33) *ctypes.Transaction {
	t.Helper()
	out := new(ctypes.Transaction)
	require.NoError(t, ctypes.Decode(marshalTx(t, tx), out))
	out.Signature = &ctypes.Signature{Ty: ctypes.EncodeSignID(secp256k1eth.ID, ctypes.EthAddressID)}
	return out
}

// checkEnvelope runs Transaction.check, the layer the mempool and block execution use,
// with minfee zero so that only the height-gated rules run.
func checkEnvelope(t *testing.T, cfg *ctypes.Chain33Config, tx *stypes.TransactionChain33, height int64) error {
	t.Helper()
	return asChain33Tx(t, tx).Check(cfg, height, 0, 0)
}

// TestEthAssemblerShapesVerify builds every shape the assembler produces with the real
// builder and requires the signature and the envelope rule to accept it at the fork.
func TestEthAssemblerShapesVerify(t *testing.T) {
	cfg, priv, pub := newTestEnv(t, false)

	cases := map[string]*etypes.Transaction{
		"transfer": newEthTx(7, &testRecipient, nil),
		"call":     newEthTx(8, &testRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb}),
		"deploy":   newEthTx(9, nil, []byte{0x60, 0x00, 0x60, 0x00}),
	}
	for name, etx := range cases {
		t.Run(name, func(t *testing.T) {
			built := assembleWith(t, cfg, etx)
			require.True(t, pub.VerifyBytes(marshalTx(t, built), signEthTx(t, priv, etx)))
			require.NoError(t, checkEnvelope(t, cfg, built, envelopeForkHeight))
		})
	}
}

func TestEthParaExecerVerifies(t *testing.T) {
	cfg, priv, pub := newTestEnv(t, true)
	etx := newEthTx(11, &testRecipient, nil)
	built := assembleWith(t, cfg, etx)
	require.Equal(t, cfg.ExecName("evm"), string(built.Execer))
	require.Equal(t, address.ExecAddress(cfg.ExecName("evm")), built.To)
	require.True(t, pub.VerifyBytes(marshalTx(t, built), signEthTx(t, priv, etx)))
	require.NoError(t, checkEnvelope(t, cfg, built, envelopeForkHeight))
}

// TestEthEnvelopeRewritesRejected mutates one envelope field at a time on a transaction
// that the assembler produced. The signature still covers the embedded Ethereum
// transaction in every case, so only the rebuild comparison stands between the mutation
// and execution.
func TestEthEnvelopeRewritesRejected(t *testing.T) {
	cfg, _, _ := newTestEnv(t, false)
	base := newEthTx(7, &testRecipient, nil)

	mutate := func(t *testing.T, change func(*stypes.TransactionChain33)) *stypes.TransactionChain33 {
		t.Helper()
		tx := assembleWith(t, cfg, base)
		change(tx)
		return tx
	}
	cases := map[string]func(*stypes.TransactionChain33){
		"execer": func(tx *stypes.TransactionChain33) { tx.Execer = []byte("user.evm.other") },
		"execer alias": func(tx *stypes.TransactionChain33) {
			tx.Execer = []byte("evmxgo")
		},
		"envelope to": func(tx *stypes.TransactionChain33) { tx.To = testAttacker.String() },
		"envelope to proxy": func(tx *stypes.TransactionChain33) {
			tx.To = strings.ToLower(testProxyAddr.String())
		},
		"fee raised":   func(tx *stypes.TransactionChain33) { tx.Fee++ },
		"fee lowered":  func(tx *stypes.TransactionChain33) { tx.Fee-- },
		"chain id":     func(tx *stypes.TransactionChain33) { tx.ChainID++ },
		"nonce":        func(tx *stypes.TransactionChain33) { tx.Nonce++ },
		"expire zero":  func(tx *stypes.TransactionChain33) { tx.Expire = 0 },
		"expire far":   func(tx *stypes.TransactionChain33) { tx.Expire = time.Now().Unix() + 86400*365 },
		"group count":  func(tx *stypes.TransactionChain33) { tx.GroupCount = 2 },
		"group header": func(tx *stypes.TransactionChain33) { tx.Header = []byte{1} },
		"group next":   func(tx *stypes.TransactionChain33) { tx.Next = []byte{1} },
		"gas price": func(tx *stypes.TransactionChain33) {
			action := decodeAction(t, tx)
			action.GasPrice = 2
			encodeAction(t, tx, action)
		},
		"gas limit": func(tx *stypes.TransactionChain33) {
			action := decodeAction(t, tx)
			action.GasLimit--
			encodeAction(t, tx, action)
		},
		"alias": func(tx *stypes.TransactionChain33) {
			action := decodeAction(t, tx)
			action.Alias = "x"
			encodeAction(t, tx, action)
		},
		"para": func(tx *stypes.TransactionChain33) {
			action := decodeAction(t, tx)
			action.Para = testAttacker.Bytes()
			encodeAction(t, tx, action)
		},
		"contract addr": func(tx *stypes.TransactionChain33) {
			action := decodeAction(t, tx)
			action.ContractAddr = testAttacker.String()
			encodeAction(t, tx, action)
		},
		"amount": func(tx *stypes.TransactionChain33) {
			action := decodeAction(t, tx)
			action.Amount++
			encodeAction(t, tx, action)
		},
		"note": func(tx *stypes.TransactionChain33) {
			action := decodeAction(t, tx)
			other := newEthTx(99, &testAttacker, nil)
			raw, err := other.MarshalBinary()
			require.NoError(t, err)
			action.Note = fmt.Sprintf("%x", raw)
			encodeAction(t, tx, action)
		},
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, checkEnvelope(t, cfg, mutate(t, change), envelopeForkHeight))
		})
	}
}

// TestEthDeployRewrapRejected covers the deployment rewrite: a pending deployment given
// call data and a proxy destination would execute as the signer on ForkProxyExec chains.
func TestEthDeployRewrapRejected(t *testing.T) {
	cfg, _, _ := newTestEnv(t, false)
	etx := newEthTx(3, nil, []byte{0x60, 0x00, 0x60, 0x00})

	tx := assembleWith(t, cfg, etx)
	action := decodeAction(t, tx)
	action.Para = testAttacker.Bytes()
	encodeAction(t, tx, action)
	tx.To = strings.ToLower(testProxyAddr.String())
	require.Error(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))
}

// TestEthEmptyDataCreationRejected covers an embedded transaction with neither To nor
// Data. The assembler cannot produce it (the coins branch dereferences To), and the
// executor would send the attached value to ContractAddr or Para.
func TestEthEmptyDataCreationRejected(t *testing.T) {
	cfg, _, _ := newTestEnv(t, false)
	etx := newEthTx(5, nil, nil)
	execAddr := address.ExecAddress(testEvmExecer)

	raw, err := etx.MarshalBinary()
	require.NoError(t, err)
	payloads := map[string]*stypes.EVMAction4Chain33{
		"coins arm": {Para: testAttacker.Bytes(), Note: fmt.Sprintf("%x", raw), ContractAddr: execAddr},
		"call arm":  {Note: fmt.Sprintf("%x", raw), ContractAddr: testAttacker.String()},
	}
	for name, action := range payloads {
		t.Run(name, func(t *testing.T) {
			payload, err := proto.Marshal(action)
			require.NoError(t, err)
			tx := &stypes.TransactionChain33{
				Execer:  []byte(testEvmExecer),
				Payload: payload,
				To:      execAddr,
				ChainID: cfg.GetChainID(),
				Expire:  time.Now().Unix() + int64(stypes.EnvelopeExpireDuration/time.Second),
			}
			require.Error(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))
		})
	}
}

// TestEthOverlongExecerRejectedWithoutPanic covers the remote crash: address.ExecPubKey
// panics on an execer longer than MaxExecNameLength. The rebuild uses the configured
// execer, so a transaction-supplied name is never dereferenced.
func TestEthOverlongExecerRejectedWithoutPanic(t *testing.T) {
	cfg, _, _ := newTestEnv(t, false)
	etx := newEthTx(3, nil, []byte{0x60, 0x00, 0x60, 0x00})
	tx := assembleWith(t, cfg, etx)
	tx.Execer = []byte(testEvmExecer + strings.Repeat("a", address.MaxExecNameLength))
	require.NotPanics(t, func() {
		require.Error(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))
	})
}

// historicalShape returns the v1.67.4 assembler output: EIP-55 checksummed To and
// ContractAddr, and the hard-coded max(gas, 1e5) fee floor instead of MinTxFeeRate.
func historicalShape(t *testing.T, cfg *ctypes.Chain33Config, etx *etypes.Transaction) *stypes.TransactionChain33 {
	t.Helper()
	tx := assembleWith(t, cfg, etx)
	action := decodeAction(t, tx)
	action.ContractAddr = ecommon.HexToAddress(action.ContractAddr).Hex()
	encodeAction(t, tx, action)
	tx.To = ecommon.HexToAddress(tx.To).Hex()
	tx.Fee = 1e5
	return tx
}

// TestEthHistoricalShapeAcceptedBeforeFork: below the fork no envelope verification runs,
// so an already-mined assembler shape keeps replaying.
func TestEthHistoricalShapeAcceptedBeforeFork(t *testing.T) {
	cfg := newTestConfig(false)
	cfg.SetMinFee(200000) // make the canonical fee differ from the legacy 1e5 floor
	priv, pub := startEnv(t, cfg)
	cfg.SetFork(ctypes.ForkEthSignEnvelope, envelopeForkHeight)
	etx := newEthTx(9, &testChecksumRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb})

	tx := historicalShape(t, cfg, etx)
	require.True(t, pub.VerifyBytes(marshalTx(t, tx), signEthTx(t, priv, etx)))
	require.NoError(t, checkEnvelope(t, cfg, tx, envelopeForkHeight-1))
}

// TestEthHistoricalShapeRejectedAtFork: the same historical shape is rejected once the
// fork is active, because it is not what the current assembler emits. This is intended:
// the fork marks the point from which the canonical envelope is required.
func TestEthHistoricalShapeRejectedAtFork(t *testing.T) {
	cfg := newTestConfig(false)
	cfg.SetMinFee(200000)
	_, _ = startEnv(t, cfg)
	cfg.SetFork(ctypes.ForkEthSignEnvelope, envelopeForkHeight)
	etx := newEthTx(9, &testChecksumRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb})

	tx := historicalShape(t, cfg, etx)
	require.Error(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))
}

// TestEthLegacyFeeRejectedAtFork: the compatibility acceptance of the old fee floor is
// gone, so a canonical envelope with the legacy fee is rejected at the fork.
func TestEthLegacyFeeRejectedAtFork(t *testing.T) {
	cfg := newTestConfig(false)
	cfg.SetMinFee(200000)
	_, _ = startEnv(t, cfg)
	cfg.SetFork(ctypes.ForkEthSignEnvelope, envelopeForkHeight)
	etx := newEthTx(9, &testRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb})

	tx := assembleWith(t, cfg, etx)
	tx.Fee = 1e5
	require.Error(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))
}

// TestEthChecksumAddressRejectedAtFork: the EIP-55 case folding is gone, so an otherwise
// canonical envelope with a checksummed address is rejected at the fork.
func TestEthChecksumAddressRejectedAtFork(t *testing.T) {
	cfg := newTestConfig(false)
	_, _ = startEnv(t, cfg)
	cfg.SetFork(ctypes.ForkEthSignEnvelope, envelopeForkHeight)
	etx := newEthTx(9, &testChecksumRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb})

	tx := assembleWith(t, cfg, etx)
	checksummed := testChecksumRecipient.Hex()
	require.NotEqual(t, strings.ToLower(checksummed), checksummed, "fixture must carry a checksummed spelling")
	action := decodeAction(t, tx)
	action.ContractAddr = checksummed
	encodeAction(t, tx, action)
	tx.To = checksummed
	require.Error(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))
}

// TestEthLegacyPrecisionAmountRejected covers the assembler before v1.67.5, which converted
// the value with a hard-coded 1e8 precision. On a chain configured with another precision
// the two conversions differ, and accepting the legacy amount would let a copied signature
// change the transferred value, so it must be rejected.
func TestEthLegacyPrecisionAmountRejected(t *testing.T) {
	cfg := ctypes.NewChain33Config(ctypes.MergeCfg("CoinPrecision=1000000\n", ctypes.GetDefaultCfgstring()))
	require.Equal(t, int64(1e6), cfg.GetCoinPrecision())
	_, _ = startEnv(t, cfg)
	cfg.SetFork(ctypes.ForkEthSignEnvelope, envelopeForkHeight)

	etx := etypes.NewTx(&etypes.DynamicFeeTx{
		ChainID:   big.NewInt(testEvmChainID),
		Nonce:     9,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       100000,
		To:        &testRecipient,
		Value:     big.NewInt(5e18),
	})
	tx := assembleWith(t, cfg, etx)
	action := decodeAction(t, tx)
	require.Equal(t, uint64(5e6), action.Amount) // configured precision

	action.Amount = 5e8 // legacy 1e8 precision
	encodeAction(t, tx, action)
	require.Error(t, checkEnvelope(t, cfg, tx, envelopeForkHeight))
}

// TestEthCarrierTyVariant is the check-skip regression: the carrier predicate must follow
// the crypto id, because the driver is selected by crypto id and the address id in the
// same value can be rewritten.
func TestEthCarrierTyVariant(t *testing.T) {
	etx := newEthTx(7, &testRecipient, nil)
	payload := ctypes.Encode(&ctypes.EVMContractAction4Chain33{Note: fmt.Sprintf("%x", mustMarshal(t, etx))})

	for addressID := int32(0); addressID <= address.MaxID; addressID++ {
		tx := &ctypes.Transaction{
			Execer:    []byte(testEvmExecer),
			Payload:   payload,
			Signature: &ctypes.Signature{Ty: ctypes.EncodeSignID(secp256k1eth.ID, addressID)},
		}
		_, ok := ctypes.EthSignedTxEnvelope(tx)
		require.True(t, ok, "address id %d", addressID)
	}
}

// TestEthTransferShapeUnderHistoricalChecks: a plain transfer is an evm payload whose Para
// holds the recipient and whose ContractAddr is the executor. The historical checks read
// the destination from there, so the shape has to keep decoding that way and a rewritten
// recipient has to fail verification.
func TestEthTransferShapeUnderHistoricalChecks(t *testing.T) {
	cfg := newTestConfig(false)
	priv, pub := startEnv(t, cfg)
	etx := newEthTx(7, &testRecipient, nil)
	tx := assembleWith(t, cfg, etx)
	action := decodeAction(t, tx)
	require.Equal(t, testRecipient.Bytes(), action.Para)
	require.Equal(t, address.ExecAddress(testEvmExecer), action.ContractAddr)
	sig := signEthTx(t, priv, etx)
	require.True(t, pub.VerifyBytes(marshalTx(t, tx), sig))

	rewritten := assembleWith(t, cfg, etx)
	ra := decodeAction(t, rewritten)
	ra.Para = testAttacker.Bytes()
	encodeAction(t, rewritten, ra)
	require.False(t, pub.VerifyBytes(marshalTx(t, rewritten), sig))
}

// TestEthAmountUsesCoinPrecision: the amount check converts the signed wei value with the
// chain's coin precision, which a node copies into the driver sub config. A driver left on
// the default precision therefore rejects a transaction assembled on such a chain, which
// pins the injection that feeds it.
func TestEthAmountUsesCoinPrecision(t *testing.T) {
	cfg := ctypes.NewChain33Config(ctypes.MergeCfg("CoinPrecision=1000000\n", ctypes.GetDefaultCfgstring()))
	require.Equal(t, int64(1e6), cfg.GetCoinPrecision())
	etx := newEthTxValue(9, &testRecipient, nil, big.NewInt(2e18))
	tx := assembleWith(t, cfg, etx)
	sig := signEthTx(t, privKeyForTest(t), etx)

	_, pub := initDriver(t, []byte(`{"evmChainID": 3999, "coinsPrecision": 1000000}`))
	require.True(t, pub.VerifyBytes(marshalTx(t, tx), sig))

	_, pubDefault := initDriver(t, []byte(`{"evmChainID": 3999}`))
	require.False(t, pubDefault.VerifyBytes(marshalTx(t, tx), sig))
}

// TestEthCarrierAddressIDRejectedAtFork: the address id sits in the signature type next to
// the crypto id, and it is not part of the signed bytes. Swapping it still verifies, but
// it derives a sender in another address format and skips the eth nonce and write
// permission rules, so from the fork on only the type the eth assembler produces is
// accepted. Below the fork the old behaviour is kept.
func TestEthCarrierAddressIDRejectedAtFork(t *testing.T) {
	cfg := newTestConfig(false)
	_, _ = startEnv(t, cfg)
	cfg.SetFork(ctypes.ForkEthSignEnvelope, envelopeForkHeight)
	etx := newEthTx(9, &testRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb})
	assembled := assembleWith(t, cfg, etx)

	require.NoError(t, asChain33Tx(t, assembled).Check(cfg, envelopeForkHeight, 0, 0))

	for addressID := int32(0); addressID <= address.MaxID; addressID++ {
		if addressID == ctypes.EthAddressID {
			continue
		}
		variant := asChain33Tx(t, assembled)
		variant.Signature.Ty = ctypes.EncodeSignID(secp256k1eth.ID, addressID)
		require.NoError(t, variant.Check(cfg, envelopeForkHeight-1, 0, 0), "address id %d below the fork", addressID)
		require.Error(t, variant.Check(cfg, envelopeForkHeight, 0, 0), "address id %d at the fork", addressID)
	}
}

// TestEthEnvelopeDuplicateFieldsRejectedWithoutFork: the nonce, amount, code and
// destination of the chain33 envelope duplicate fields of the signed Ethereum
// transaction, and they were checked at signature verification long before the fork-gated
// rebuild existed. They must keep rejecting rewrites below the fork, otherwise an
// upgrade would leave a window in which forging is easier than it was before.
func TestEthEnvelopeDuplicateFieldsRejectedWithoutFork(t *testing.T) {
	cfg := newTestConfig(false)
	priv, pub := startEnv(t, cfg)
	etx := newEthTx(9, &testRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb})
	sig := signEthTx(t, priv, etx)
	require.True(t, pub.VerifyBytes(marshalTx(t, assembleWith(t, cfg, etx)), sig))

	cases := map[string]func(*stypes.EVMAction4Chain33){
		"amount": func(a *stypes.EVMAction4Chain33) { a.Amount++ },
		"code":   func(a *stypes.EVMAction4Chain33) { a.Para = []byte{0x01, 0x02} },
		"to":     func(a *stypes.EVMAction4Chain33) { a.ContractAddr = testAttacker.Hex() },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			tx := assembleWith(t, cfg, etx)
			action := decodeAction(t, tx)
			mutate(action)
			encodeAction(t, tx, action)
			require.False(t, pub.VerifyBytes(marshalTx(t, tx), sig),
				"a rewritten envelope must not pass signature verification")
		})
	}
}

func mustMarshal(t *testing.T, etx *etypes.Transaction) []byte {
	t.Helper()
	raw, err := etx.MarshalBinary()
	require.NoError(t, err)
	return raw
}
