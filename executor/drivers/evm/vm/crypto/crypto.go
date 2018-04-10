package crypto

import (
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/rlp"
	"math/big"
)

var (
	// FIXME 这个常量的定义目前还不清楚，待确认
	secp256k1_N, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)
)

// ValidateSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateSignatureValues(v byte, r, s *big.Int) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256k1_N) < 0 && s.Cmp(secp256k1_N) < 0 && (v == 0 || v == 1)
}

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	//return secp256k1.RecoverPubkey(hash, sig)
	// TODO 实现方式待验证
	publicKey, result, err := btcec.RecoverCompact(btcec.S256(), sig, hash)
	if err != nil {
		return nil, err
	}
	if !result {
		return nil, errors.New("recover error!")
	}

	return publicKey.SerializeUncompressed(), nil
}

// Creates an ethereum address given the bytes and the nonce
func CreateAddress(b common.Address, nonce uint64) common.Address {
	data, _ := rlp.EncodeToBytes([]interface{}{b, nonce})
	return common.BytesToAddress(Keccak256(data)[12:])
}
