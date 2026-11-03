package types

import (
	"crypto/ecdsa"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/common"
	chain33crypto "github.com/33cn/chain33/common/crypto"
	_ "github.com/33cn/chain33/system/address"
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	ctypes "github.com/33cn/chain33/types"

	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const assembleTestKey = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"

// assembleEvmChainID is evmChainID from the default chain33 config.
const assembleEvmChainID = 3999

var (
	assembleRecipient = ecommon.HexToAddress("0x2222222222222222222222222222222222222222")
	assembleProxy     = ecommon.HexToAddress("0x0000000000000000000000000000000000200005")
)

func newAssembleConfig(t *testing.T, para bool) *ctypes.Chain33Config {
	t.Helper()
	cfg := ctypes.NewChain33Config(ctypes.GetDefaultCfgstring())
	if para {
		cfg.SetTitleOnlyForTest("user.p.para.")
	}
	chain33crypto.Init(cfg.GetModuleConfig().Crypto, map[string][]byte{
		secp256k1eth.Name: cfg.EthCryptoSubConfig(cfg.GetSubConfig().Crypto[secp256k1eth.Name]),
	})
	return cfg
}

func newAssembleKey(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()
	key, err := common.FromHex(assembleTestKey)
	require.NoError(t, err)
	priv, err := ethcrypto.ToECDSA(key)
	require.NoError(t, err)
	return priv, ethcrypto.FromECDSAPub(&priv.PublicKey)
}

func newAssembleEthTx(t *testing.T, nonce uint64, to *ecommon.Address, data []byte) *etypes.Transaction {
	t.Helper()
	return etypes.NewTx(&etypes.DynamicFeeTx{
		ChainID:   big.NewInt(assembleEvmChainID),
		Nonce:     nonce,
		GasTipCap: big.NewInt(1),
		GasFeeCap: big.NewInt(1),
		Gas:       100000,
		To:        to,
		Value:     big.NewInt(0),
		Data:      data,
	})
}

func signAssembleEthTx(t *testing.T, priv *ecdsa.PrivateKey, etx *etypes.Transaction) []byte {
	t.Helper()
	sig, err := ethcrypto.Sign(etypes.NewLondonSigner(etx.ChainId()).Hash(etx).Bytes(), priv)
	require.NoError(t, err)
	return sig
}

// TestAssembleChain33TxVerifies assembles through the production entry point and checks
// the signature through the production verifier. If the assembler and the verifier read
// one different parameter or rule, the signature check fails here.
func TestAssembleChain33TxVerifies(t *testing.T) {
	priv, pub := newAssembleKey(t)
	inner := &ctypes.Transaction{
		Execer:  []byte("coins"),
		Payload: []byte{0x01},
		To:      assembleRecipient.String(),
		Nonce:   1,
	}
	cases := map[string]*etypes.Transaction{
		"transfer": newAssembleEthTx(t, 1, &assembleRecipient, nil),
		"call":     newAssembleEthTx(t, 2, &assembleRecipient, []byte{0xa9, 0x05, 0x9c, 0xbb}),
		"deploy":   newAssembleEthTx(t, 3, nil, []byte{0x60, 0x00, 0x60, 0x00}),
		"proxy":    newAssembleEthTx(t, 4, &assembleProxy, ctypes.Encode(inner)),
	}
	for name, etx := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := newAssembleConfig(t, false)
			tx := AssembleChain33Tx(etx, signAssembleEthTx(t, priv, etx), pub, cfg)
			require.NotNil(t, tx)
			require.True(t, tx.CheckSign(0))
		})
	}
}

// TestAssembleChain33TxVerifiesOnParaChain covers cfg.ExecName("evm") on a parallel
// chain, where the envelope To is the exec address rather than the embedded address.
func TestAssembleChain33TxVerifiesOnParaChain(t *testing.T) {
	priv, pub := newAssembleKey(t)
	cfg := newAssembleConfig(t, true)
	etx := newAssembleEthTx(t, 5, &assembleRecipient, nil)
	tx := AssembleChain33Tx(etx, signAssembleEthTx(t, priv, etx), pub, cfg)
	require.NotNil(t, tx)
	require.Equal(t, cfg.ExecName("evm"), string(tx.Execer))
	require.True(t, tx.CheckSign(0))
}
