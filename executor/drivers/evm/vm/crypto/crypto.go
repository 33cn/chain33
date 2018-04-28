package crypto

import (
	"errors"
	"github.com/btcsuite/btcd/btcec"
	"math/big"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
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

// 随机生成一个新的地址，给新创建的合约地址使用
func RandomAddress() *common.Address {
	c, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		return nil
	}
	key, err := c.GenKey()
	if err != nil {
		return nil
	}

	acc := account.PubKeyToAddress(key.PubKey().Bytes())
	addr := common.StringToAddress(acc.String())
	return &addr
}