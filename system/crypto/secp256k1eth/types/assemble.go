package types

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/33cn/chain33/common/address"
	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"

	"google.golang.org/protobuf/proto"
)

const (
	// EnvelopeExpireDuration is the lifetime AssembleChain33Tx gives the assembled
	// transaction. The assembler always expresses it as a wall-clock expiry: the
	// TxHeight branch of SetExpire only applies to durations above TxHeightFlag, which a
	// three hour duration never reaches.
	EnvelopeExpireDuration = 3 * time.Hour
	// EnvelopeExpireSkew bounds the clock difference between the node that assembles a
	// transaction and the node that verifies it.
	EnvelopeExpireSkew = 10 * time.Minute
)

// AssembleParams carries everything AssembleChain33Tx reads from the node configuration.
// The assembler and the verifier both fill it from the same chain config, so the rules
// cannot drift apart.
type AssembleParams struct {
	ChainID       int32
	EVMExecer     string
	MinTxFeeRate  int64
	CoinPrecision int64
	IsPara        bool
	Now           time.Time
}

// assembledTx is the canonical transaction together with its decoded payload.
type assembledTx struct {
	tx     *TransactionChain33
	action *EVMAction4Chain33
}

// ethValueToCoinAmount converts an Ethereum value to chain33 coin units at the configured
// precision and rejects values that do not fit uint64.
func ethValueToCoinAmount(value *big.Int, coinPrecision int64) (uint64, error) {
	if value == nil {
		return 0, nil
	}
	if coinPrecision <= 0 || coinPrecision > 1e18 {
		return 0, fmt.Errorf("invalid coin precision %d", coinPrecision)
	}
	amount := PrecisionEth2Coins(value, coinPrecision)
	if !amount.IsInt64() || amount.Sign() < 0 {
		return 0, fmt.Errorf("value %s out of range at precision %d", value, coinPrecision)
	}
	return uint64(amount.Int64()), nil
}

// PrecisionEth2Coins converts an Ethereum value (1e18 units) to chain33 coin units.
func PrecisionEth2Coins(ethValue *big.Int, coinPrecision int64) *big.Int {
	ethUnit := big.NewInt(1e18)
	return new(big.Int).Div(ethValue, ethUnit.Div(ethUnit, big.NewInt(coinPrecision)))
}

// AssembleChain33Tx rebuilds the canonical chain33 transaction for an embedded Ethereum
// transaction. This is the single producer of the eth-signature envelope: the JSON-RPC
// assembler uses it, and the signature verifier rebuilds the transaction from the
// embedded Ethereum transaction with the same function and requires equality.
func AssembleChain33Tx(etx *etypes.Transaction, p AssembleParams) (*TransactionChain33, error) {
	a, err := assemble(etx, p)
	if err != nil {
		return nil, err
	}
	return a.tx, nil
}

func assemble(etx *etypes.Transaction, p AssembleParams) (*assembledTx, error) {
	if etx == nil {
		return nil, errors.New("nil embedded transaction")
	}
	if p.EVMExecer == "" {
		return nil, errors.New("evm execer not configured")
	}
	rawData, err := etx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("marshal embedded transaction: %w", err)
	}
	amount, err := ethValueToCoinAmount(etx.Value(), p.CoinPrecision)
	if err != nil {
		return nil, err
	}

	action := &EVMAction4Chain33{
		Amount:   amount,
		GasLimit: etx.Gas(),
		GasPrice: 1,
		Note:     hex.EncodeToString(rawData),
	}
	var to string
	if len(etx.Data()) != 0 { // contract operation
		packdata := etx.Data()
		if etx.To() == nil || len(etx.To().Bytes()) == 0 {
			// contract deploy: the payload carries the code and names the exec address.
			// The EVM executor reads ContractAddr as the receiver, so an empty value makes
			// it reject the deployment after the fee has already been charged.
			action.Code = packdata
			to = address.ExecAddress(p.EVMExecer)
			action.ContractAddr = to
		} else {
			// contract call: the payload carries the calldata and names the callee
			action.Para = packdata
			to = strings.ToLower(etx.To().String())
			action.ContractAddr = to
		}
	} else { // coins operation
		if etx.To() == nil {
			// the assembler dereferences To; this shape never had a producer
			return nil, errors.New("transfer without a destination address")
		}
		to = strings.ToLower(etx.To().String())
		action.Para = ecommon.FromHex(to)
		action.ContractAddr = address.ExecAddress(p.EVMExecer)
	}
	if p.IsPara {
		to = address.ExecAddress(p.EVMExecer)
	}

	payload, err := proto.MarshalOptions{Deterministic: true}.Marshal(action)
	if err != nil {
		return nil, fmt.Errorf("marshal evm action: %w", err)
	}
	fee := int64(etx.Gas())
	if fee < p.MinTxFeeRate {
		fee = p.MinTxFeeRate
	}
	tx := &TransactionChain33{
		ChainID: p.ChainID,
		To:      to,
		Execer:  []byte(p.EVMExecer),
		Payload: payload,
		Fee:     fee,
		Nonce:   int64(etx.Nonce()),
		Expire:  p.Now.Unix() + int64(EnvelopeExpireDuration/time.Second),
	}
	return &assembledTx{tx: tx, action: action}, nil
}
