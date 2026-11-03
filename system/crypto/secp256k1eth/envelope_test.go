package secp256k1eth_test

import (
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/require"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/address"
	chain33crypto "github.com/33cn/chain33/common/crypto"
	_ "github.com/33cn/chain33/system/address" // register the btc and eth address drivers
	"github.com/33cn/chain33/system/crypto/secp256k1eth"
	stypes "github.com/33cn/chain33/system/crypto/secp256k1eth/types"

	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const (
	testEvmChainID = 3999
	testEvmExecer  = "evm"
)

var (
	testPrivKeyHex = "CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944"
	testAttacker   = ecommon.HexToAddress("0x1111111111111111111111111111111111111111")
	testRecipient  = ecommon.HexToAddress("0x2222222222222222222222222222222222222222")
)

// initEthEnv pins the chain id and the coins precision the verifier reads, so the tests
// do not depend on whatever another test in this package left behind.
func initEthEnv(t *testing.T) (*ecdsa.PrivateKey, chain33crypto.PubKey) {
	t.Helper()
	chain33crypto.Init(&chain33crypto.Config{}, map[string][]byte{
		secp256k1eth.Name: []byte(`{"evmChainID": 3999, "coinsPrecision": 100000000}`),
	})
	require.Equal(t, int64(testEvmChainID), secp256k1eth.GetEvmChainID())

	key, err := common.FromHex(testPrivKeyHex)
	require.NoError(t, err)
	priv, err := ethcrypto.ToECDSA(key)
	require.NoError(t, err)

	driver := secp256k1eth.Driver{}
	pub, err := driver.PubKeyFromBytes(ethcrypto.FromECDSAPub(&priv.PublicKey))
	require.NoError(t, err)
	return priv, pub
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

func marshalEthTx(t *testing.T, etx *etypes.Transaction) []byte {
	t.Helper()
	raw, err := etx.MarshalBinary()
	require.NoError(t, err)
	return raw
}

func marshalChain33Tx(t *testing.T, execer string, payload proto.Message, nonce int64, txTo string) []byte {
	t.Helper()
	body, err := proto.Marshal(payload)
	require.NoError(t, err)
	msg, err := proto.Marshal(&stypes.TransactionChain33{
		Execer:  []byte(execer),
		Payload: body,
		Nonce:   nonce,
		To:      txTo,
	})
	require.NoError(t, err)
	return msg
}

func ethExecAddr(t *testing.T, execer string) string {
	t.Helper()
	driver, err := address.LoadDriver(2, -1)
	require.NoError(t, err)
	return driver.PubKeyToAddr(address.ExecPubKey(execer))
}

func TestEthPaymentStillVerifies(t *testing.T) {
	priv, pub := initEthEnv(t)

	etx := newEthTx(7, &testRecipient, nil)
	action := &stypes.EVMAction4Chain33{
		Amount:       0,
		Para:         testRecipient.Bytes(),
		Note:         ecommon.Bytes2Hex(marshalEthTx(t, etx)),
		ContractAddr: address.ExecAddress(testEvmExecer), // coins transfer: ContractAddr is the exec address
	}
	msg := marshalChain33Tx(t, testEvmExecer, action, int64(etx.Nonce()), address.ExecAddress(testEvmExecer))

	require.True(t, pub.VerifyBytes(msg, signEthTx(t, priv, etx)))
}

func TestEthContractCallStillVerifies(t *testing.T) {
	priv, pub := initEthEnv(t)

	callData := []byte{0xa9, 0x05, 0x9c, 0xbb}
	etx := newEthTx(9, &testRecipient, callData)
	action := &stypes.EVMAction4Chain33{
		Amount:       0,
		Para:         callData,
		Note:         ecommon.Bytes2Hex(marshalEthTx(t, etx)),
		ContractAddr: testRecipient.String(), // calls are not subject to the deployment check
	}
	msg := marshalChain33Tx(t, testEvmExecer, action, int64(etx.Nonce()), address.ExecAddress(testEvmExecer))

	require.True(t, pub.VerifyBytes(msg, signEthTx(t, priv, etx)))
}

func TestEthCoinsRewrapRejected(t *testing.T) {
	priv, pub := initEthEnv(t)

	// A published eth-channel payment: pays testRecipient.
	etx := newEthTx(7, &testRecipient, nil)
	raw := marshalEthTx(t, etx)

	// The same signature re-wrapped as a coins-exec transaction whose envelope To pays
	// the attacker. The payload keeps the original recipient so every pre-existing check
	// (nonce, amount, payload To) still passes; only the envelope changed.
	payload := &stypes.CoinsActionChain33{
		Value: &stypes.CoinsActionChain33_Transfer{Transfer: &stypes.AssetsTransferChain33{
			Amount: 0,
			Note:   raw,
			To:     testRecipient.String(),
		}},
	}
	msg := marshalChain33Tx(t, "coins", payload, int64(etx.Nonce()), testAttacker.String())

	require.False(t, pub.VerifyBytes(msg, signEthTx(t, priv, etx)))
}

func TestEthDeploymentExecAddrVerifies(t *testing.T) {
	priv, pub := initEthEnv(t)

	code := []byte{0x60, 0x00, 0x60, 0x00}
	etx := newEthTx(3, nil, code)
	raw := marshalEthTx(t, etx)

	ethAddr := ethExecAddr(t, testEvmExecer)
	cases := map[string]string{
		"btc spelling":   address.ExecAddress(testEvmExecer),
		"eth spelling":   ethAddr,
		"eth upper":      "0x" + strings.ToUpper(ethAddr[2:]),
		"eth without 0x": ethAddr[2:],
	}
	for name, contractAddr := range cases {
		t.Run(name, func(t *testing.T) {
			action := &stypes.EVMAction4Chain33{
				Amount:       0,
				Code:         code,
				Note:         ecommon.Bytes2Hex(raw),
				ContractAddr: contractAddr,
			}
			msg := marshalChain33Tx(t, testEvmExecer, action, int64(etx.Nonce()), address.ExecAddress(testEvmExecer))
			require.True(t, pub.VerifyBytes(msg, signEthTx(t, priv, etx)))
		})
	}
}

func TestEthDeploymentAttackerAddrRejected(t *testing.T) {
	priv, pub := initEthEnv(t)

	code := []byte{0x60, 0x00, 0x60, 0x00}
	etx := newEthTx(3, nil, code)
	action := &stypes.EVMAction4Chain33{
		Amount:       0,
		Code:         code,
		Note:         ecommon.Bytes2Hex(marshalEthTx(t, etx)),
		ContractAddr: testAttacker.String(), // makes the executor call instead of create
	}
	msg := marshalChain33Tx(t, testEvmExecer, action, int64(etx.Nonce()), address.ExecAddress(testEvmExecer))

	require.False(t, pub.VerifyBytes(msg, signEthTx(t, priv, etx)))
}

// TestEthEmptyDataCreationRejected covers the one shape with no To in the embedded tx and
// no Data: the executor would not create a contract, it would send the attached value to
// ContractAddr or Para. The assembler cannot produce it, so it is always rejected.
func TestEthEmptyDataCreationRejected(t *testing.T) {
	priv, pub := initEthEnv(t)

	etx := newEthTx(5, nil, nil)
	raw := marshalEthTx(t, etx)
	execAddr := address.ExecAddress(testEvmExecer)

	cases := map[string]*stypes.EVMAction4Chain33{
		// coins arm: ContractAddr is the exec address, so the value goes to Para.
		"coins arm": {
			Amount:       0,
			Para:         testAttacker.Bytes(),
			Note:         ecommon.Bytes2Hex(raw),
			ContractAddr: execAddr,
		},
		// call arm: ContractAddr is the call target and Para is empty.
		"call arm": {
			Amount:       0,
			Note:         ecommon.Bytes2Hex(raw),
			ContractAddr: testAttacker.String(),
		},
	}
	for name, action := range cases {
		t.Run(name, func(t *testing.T) {
			msg := marshalChain33Tx(t, testEvmExecer, action, int64(etx.Nonce()), execAddr)
			require.False(t, pub.VerifyBytes(msg, signEthTx(t, priv, etx)))
		})
	}
}
