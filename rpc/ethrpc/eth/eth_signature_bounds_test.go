package eth

// Regression test: eth_sendRawTransaction must reject oversized R/S instead of
// indexing the 65-byte signature buffer with a negative offset.
// Before the fix len(r.Bytes()) > 32 made copy(sig[32-len(r.Bytes()):32], ...) panic.

import (
	"encoding/hex"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestSendRawTransactionRejectsOversizedSignature(t *testing.T) {
	h := &ethHandler{evmChainID: 1}
	to := common.HexToAddress("0x0000000000000000000000000000000000000001")
	// 33 bytes, high bit set so RLP encoding has no leading zero.
	oversized := new(big.Int).SetBytes(append([]byte{0x80}, make([]byte, 32)...))

	type legacyTx struct {
		Nonce    uint64
		GasPrice *big.Int
		Gas      uint64
		To       *common.Address
		Value    *big.Int
		Data     []byte
		V        *big.Int
		R        *big.Int
		S        *big.Int
	}
	raw, err := rlp.EncodeToBytes(&legacyTx{
		Nonce: 0, GasPrice: big.NewInt(1), Gas: 21000, To: &to,
		Value: big.NewInt(0), Data: nil, V: big.NewInt(37),
		R: oversized, S: big.NewInt(1),
	})
	if err != nil {
		t.Fatalf("encode legacy tx: %v", err)
	}

	var returned error
	panicked := func() (p interface{}) {
		defer func() { p = recover() }()
		_, returned = h.SendRawTransaction(hex.EncodeToString(raw))
		return nil
	}()
	if panicked != nil {
		t.Fatalf("oversized R/S must not panic: %v", panicked)
	}
	if returned == nil || !strings.Contains(returned.Error(), "invalid signature length") {
		t.Fatalf("want invalid signature length error, got: %v", returned)
	}
}
