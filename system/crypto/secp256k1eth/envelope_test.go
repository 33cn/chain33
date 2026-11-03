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
)

var (
	testPrivKeyHex = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	testAttacker   = ecommon.HexToAddress("0x1111111111111111111111111111111111111111")
	testRecipient  = ecommon.HexToAddress("0x2222222222222222222222222222222222222222")
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

// startEnv initializes the driver exactly as a node does: the chain parameters are
// copied into the [crypto.sub.secp256k1eth] sub config by Chain33Config.EthCryptoSubConfig.
func startEnv(t *testing.T, cfg *ctypes.Chain33Config) (*ecdsa.PrivateKey, chain33crypto.PubKey) {
	t.Helper()
	oldChainID := secp256k1eth.GetEvmChainID()
	sub := cfg.GetSubConfig().Crypto[secp256k1eth.Name]
	chain33crypto.Init(&chain33crypto.Config{}, map[string][]byte{
		secp256k1eth.Name: cfg.EthCryptoSubConfig(sub),
	})
	t.Cleanup(func() {
		chain33crypto.Init(&chain33crypto.Config{}, map[string][]byte{
			secp256k1eth.Name: []byte(fmt.Sprintf(`{"evmChainID": %d, "coinsPrecision": 100000000}`, oldChainID)),
		})
	})

	key, err := common.FromHex(testPrivKeyHex)
	require.NoError(t, err)
	priv, err := ethcrypto.ToECDSA(key)
	require.NoError(t, err)
	driver := secp256k1eth.Driver{}
	pub, err := driver.PubKeyFromBytes(ethcrypto.FromECDSAPub(&priv.PublicKey))
	require.NoError(t, err)
	return priv, pub
}

func newTestEnv(t *testing.T, para bool) (*ctypes.Chain33Config, *ecdsa.PrivateKey, chain33crypto.PubKey) {
	t.Helper()
	cfg := newTestConfig(para)
	priv, pub := startEnv(t, cfg)
	return cfg, priv, pub
}

func newEthTx(nonce uint64, to *ecommon.Address, data []byte) *etypes.Transaction {
	return etypes.NewTx(&etypes.DynamicFeeTx{
		ChainID:   big.NewInt(testEvmChainID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       100000,
		To:        to,
		Value:     big.NewInt(0),
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

// TestEthAssemblerShapesVerify builds every shape the assembler produces with the real
// builder and requires the verifier to accept it.
func TestEthAssemblerShapesVerify(t *testing.T) {
	cfg, priv, pub := newTestEnv(t, false)

	cases := map[string]*etypes.Transaction{
		"transfer": newEthTx(7, &testRecipient, nil),
		"call":     newEthTx(8, &testRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb}),
		"deploy":   newEthTx(9, nil, []byte{0x60, 0x00, 0x60, 0x00}),
	}
	for name, etx := range cases {
		t.Run(name, func(t *testing.T) {
			msg := marshalTx(t, assembleWith(t, cfg, etx))
			require.True(t, pub.VerifyBytes(msg, signEthTx(t, priv, etx)))
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
}

// TestEthEnvelopeRewritesRejected mutates one envelope field at a time on a transaction
// that the assembler produced. The signature still covers the embedded Ethereum
// transaction in every case, so only the rebuild comparison stands between the mutation
// and execution.
func TestEthEnvelopeRewritesRejected(t *testing.T) {
	cfg, priv, pub := newTestEnv(t, false)
	base := newEthTx(7, &testRecipient, nil)
	sig := signEthTx(t, priv, base)

	mutate := func(t *testing.T, change func(*stypes.TransactionChain33)) []byte {
		t.Helper()
		tx := assembleWith(t, cfg, base)
		change(tx)
		return marshalTx(t, tx)
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
			require.False(t, pub.VerifyBytes(mutate(t, change), sig))
		})
	}
}

// TestEthDeployRewrapRejected covers the deployment rewrite: a pending deployment given
// call data and a proxy destination would execute as the signer on ForkProxyExec chains.
func TestEthDeployRewrapRejected(t *testing.T) {
	cfg, priv, pub := newTestEnv(t, false)
	etx := newEthTx(3, nil, []byte{0x60, 0x00, 0x60, 0x00})
	sig := signEthTx(t, priv, etx)

	tx := assembleWith(t, cfg, etx)
	action := decodeAction(t, tx)
	action.Para = testAttacker.Bytes()
	encodeAction(t, tx, action)
	tx.To = strings.ToLower(testProxyAddr.String())
	require.False(t, pub.VerifyBytes(marshalTx(t, tx), sig))
}

// TestEthEmptyDataCreationRejected covers an embedded transaction with neither To nor
// Data. The assembler cannot produce it (the coins branch dereferences To), and the
// executor would send the attached value to ContractAddr or Para.
func TestEthEmptyDataCreationRejected(t *testing.T) {
	cfg, priv, pub := newTestEnv(t, false)
	etx := newEthTx(5, nil, nil)
	sig := signEthTx(t, priv, etx)
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
			require.False(t, pub.VerifyBytes(marshalTx(t, tx), sig))
		})
	}
}

// TestEthOverlongExecerRejectedWithoutPanic covers the remote crash: address.ExecPubKey
// panics on an execer longer than MaxExecNameLength.
func TestEthOverlongExecerRejectedWithoutPanic(t *testing.T) {
	cfg, priv, pub := newTestEnv(t, false)
	etx := newEthTx(3, nil, []byte{0x60, 0x00, 0x60, 0x00})
	tx := assembleWith(t, cfg, etx)
	tx.Execer = []byte(testEvmExecer + strings.Repeat("a", address.MaxExecNameLength))
	msg := marshalTx(t, tx)
	sig := signEthTx(t, priv, etx)
	require.NotPanics(t, func() {
		require.False(t, pub.VerifyBytes(msg, sig))
	})
}

// TestEthHistoricalShapesVerify pins the two assembler eras: v1.67.4 wrote the EIP-55
// checksummed To/ContractAddr and the hard-coded fee floor of 1e5; v1.68.0 lower-cased the
// addresses and read MinTxFeeRate. Both must keep verifying, or replaying old blocks
// would fork.
func TestEthHistoricalShapesVerify(t *testing.T) {
	cfg := newTestConfig(false)
	cfg.SetMinFee(200000) // make the two fee rules differ
	priv, pub := startEnv(t, cfg)
	etx := newEthTx(9, &testRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb})
	sig := signEthTx(t, priv, etx)

	tx := assembleWith(t, cfg, etx)
	action := decodeAction(t, tx)
	action.ContractAddr = ecommon.HexToAddress(action.ContractAddr).Hex() // EIP-55 spelling
	encodeAction(t, tx, action)
	tx.To = ecommon.HexToAddress(tx.To).Hex()
	tx.Fee = 1e5 // legacy max(gas, 1e5) floor; gas is below it
	require.True(t, pub.VerifyBytes(marshalTx(t, tx), sig))
}

// TestEthLegacyPrecisionAmountRejected covers the assembler before v1.67.5, which converted
// the value with a hard-coded 1e8 precision. On a chain configured with another precision
// the two conversions differ, and accepting the legacy amount would let a copied signature
// change the transferred value, so it must be rejected.
func TestEthLegacyPrecisionAmountRejected(t *testing.T) {
	cfg := ctypes.NewChain33Config(ctypes.MergeCfg("CoinPrecision=1000000\n", ctypes.GetDefaultCfgstring()))
	require.Equal(t, int64(1e6), cfg.GetCoinPrecision())
	priv, pub := startEnv(t, cfg)

	etx := etypes.NewTx(&etypes.DynamicFeeTx{
		ChainID:   big.NewInt(testEvmChainID),
		Nonce:     9,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       100000,
		To:        &testRecipient,
		Value:     big.NewInt(5e18),
	})
	sig := signEthTx(t, priv, etx)
	tx := assembleWith(t, cfg, etx)
	action := decodeAction(t, tx)
	require.Equal(t, uint64(5e6), action.Amount) // configured precision

	action.Amount = 5e8 // legacy 1e8 precision
	encodeAction(t, tx, action)
	require.False(t, pub.VerifyBytes(marshalTx(t, tx), sig))
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

func mustMarshal(t *testing.T, etx *etypes.Transaction) []byte {
	t.Helper()
	raw, err := etx.MarshalBinary()
	require.NoError(t, err)
	return raw
}
