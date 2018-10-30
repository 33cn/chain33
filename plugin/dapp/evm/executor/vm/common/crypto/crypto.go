package crypto

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/btcsuite/btcd/btcec"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/types"
	"golang.org/x/crypto/sha3"
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
	c, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		return nil
	}
	key, err := c.GenKey()
	if err != nil {
		return nil
	}

	acc := address.PubKeyToAddress(key.PubKey().Bytes())
	ret := common.StringToAddress(address.ExecAddress(acc.String()))
	return ret
}

// Keccak256 计算并返回 Keccak256 哈希
func Keccak256(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}
