package types

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	etypes "github.com/ethereum/go-ethereum/core/types"

	"google.golang.org/protobuf/proto"
)

// expireBoundSeconds mirrors types.ExpireBound, which cannot be imported here because
// that package imports this one. A value at or below it is a block height, not a time.
const expireBoundSeconds = int64(1e9)

// VerifyAssembledTx requires tx to be exactly the transaction the assembler produces for
// etx. Only the expiry is compared as a window, because it is derived from the assembler
// clock rather than from the signed Ethereum transaction.
func VerifyAssembledTx(tx *TransactionChain33, etx *etypes.Transaction, p AssembleParams) error {
	if tx == nil {
		return errors.New("nil chain33 transaction")
	}
	if etx == nil {
		return errors.New("nil embedded transaction")
	}
	var action EVMAction4Chain33
	if err := proto.Unmarshal(tx.Payload, &action); err != nil {
		return fmt.Errorf("decode evm payload: %w", err)
	}
	want, err := assemble(etx, p)
	if err != nil {
		return err
	}

	if string(tx.Execer) != string(want.tx.Execer) {
		return fmt.Errorf("execer %q, want %q", tx.Execer, want.tx.Execer)
	}
	if tx.ChainID != want.tx.ChainID {
		return fmt.Errorf("chainID %d, want %d", tx.ChainID, want.tx.ChainID)
	}
	if tx.Nonce != want.tx.Nonce {
		return fmt.Errorf("nonce %d, want %d", tx.Nonce, want.tx.Nonce)
	}
	if tx.GroupCount != 0 || len(tx.Header) != 0 || len(tx.Next) != 0 {
		return fmt.Errorf("group fields set: count %d header %d next %d", tx.GroupCount, len(tx.Header), len(tx.Next))
	}
	if tx.Fee != want.tx.Fee {
		return fmt.Errorf("fee %d, want %d", tx.Fee, want.tx.Fee)
	}
	if !validExpire(tx.Expire, p.Now) {
		return fmt.Errorf("expire %d outside the assembler window", tx.Expire)
	}
	if !sameEnvelopeAddress(want.tx.To, tx.To) {
		return fmt.Errorf("envelope To %q, want %q", tx.To, want.tx.To)
	}
	if action.GasPrice != want.action.GasPrice {
		return fmt.Errorf("gasPrice %d, want %d", action.GasPrice, want.action.GasPrice)
	}
	if action.GasLimit != want.action.GasLimit {
		return fmt.Errorf("gasLimit %d, want %d", action.GasLimit, want.action.GasLimit)
	}
	if action.Amount != want.action.Amount {
		return fmt.Errorf("amount %d, want %d", action.Amount, want.action.Amount)
	}
	if action.Alias != want.action.Alias {
		return fmt.Errorf("alias %q, want %q", action.Alias, want.action.Alias)
	}
	if !bytes.Equal(action.Code, want.action.Code) {
		return errors.New("code mismatch")
	}
	if !bytes.Equal(action.Para, want.action.Para) {
		return errors.New("para mismatch")
	}
	if action.Note != want.action.Note {
		return errors.New("note mismatch")
	}
	if !sameEnvelopeAddress(want.action.ContractAddr, action.ContractAddr) {
		return fmt.Errorf("contract address %q, want %q", action.ContractAddr, want.action.ContractAddr)
	}
	return nil
}

// sameEnvelopeAddress compares an address with the assembler's canonical spelling. The
// optional 0x prefix is tolerated because it does not select a different address; the hex
// digits themselves are compared exactly, including their case.
func sameEnvelopeAddress(expected, addr string) bool {
	return strings.TrimPrefix(expected, "0x") == strings.TrimPrefix(addr, "0x")
}

// validExpire applies the upper bound of the assembler window: the expiry was computed
// from the assembler clock, so it may sit up to the lifetime plus the skew ahead of this
// node's clock and can never be zero or a block height. The lower bound (expiry not yet
// reached) is enforced by the block and mempool expiry checks; applying it here would
// reject every already-mined transaction when an old block is re-validated.
func validExpire(expire int64, now time.Time) bool {
	if expire <= expireBoundSeconds { // 0 or a block height: never assembler-produced
		return false
	}
	upper := now.Unix() + int64(EnvelopeExpireDuration/time.Second) + int64(EnvelopeExpireSkew/time.Second)
	return expire <= upper
}

// DecodeEmbeddedTx decodes the raw Ethereum transaction carried in a payload Note.
func DecodeEmbeddedTx(note string) (*etypes.Transaction, error) {
	raw, err := hex.DecodeString(note)
	if err != nil {
		return nil, fmt.Errorf("decode embedded transaction hex: %w", err)
	}
	etx := new(etypes.Transaction)
	if err := etx.UnmarshalBinary(raw); err != nil {
		return nil, fmt.Errorf("decode embedded transaction: %w", err)
	}
	return etx, nil
}
