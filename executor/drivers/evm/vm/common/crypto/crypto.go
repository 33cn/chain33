package crypto

import (
	"math/big"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/account"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
	"golang.org/x/crypto/sha3"
	"github.com/btcsuite/btcd/btcec"
	"crypto/ecdsa"
)

// 校验签名信息是否正确
func ValidateSignatureValues(r, s *big.Int) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	return true
}

// 根据压缩消息和签名，返回未压缩的公钥信息
func Ecrecover(hash, sig []byte) ([]byte, error) {
	pub, err := SigToPub(hash, sig)
	if err != nil {
		return nil, err
	}
	bytes := (*btcec.PublicKey)(pub).SerializeUncompressed()
	return bytes, err
}

// 根据签名返回公钥信息
func SigToPub(hash, sig []byte) (*ecdsa.PublicKey, error) {
	btcsig := make([]byte, 65)
	btcsig[0] = sig[64] + 27
	copy(btcsig[1:], sig)

	pub, _, err := btcec.RecoverCompact(btcec.S256(), btcsig, hash)
	return (*ecdsa.PublicKey)(pub), err
}


// 随机生成一个新的地址，给新创建的合约地址使用
func RandomContractAddress() *common.Address {
	c, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		return nil
	}
	key, err := c.GenKey()
	if err != nil {
		return nil
	}

	acc := account.PubKeyToAddress(key.PubKey().Bytes())
	ret := common.StringToAddress(account.ExecAddress(acc.String()).String())
	return ret
}

// Keccak256 计算并返回 Keccak256 哈希
// 直接使用了sha3内置的方法逻辑，和以太坊的逻辑不同 (FIXME 需要更新golang.org/x/crypto/sha3包，使用最新的方法NewLegacyKeccak256，以保证和以太坊逻辑相同)
func Keccak256(data ...[]byte) []byte {
	d := sha3.New256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
