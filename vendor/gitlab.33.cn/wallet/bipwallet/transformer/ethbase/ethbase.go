//转换基于以太坊地址规则的币种
//ETH、ETC

package ethbase

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/haltingstate/secp256k1-go"
)

type EthBaseTransformer struct{}

// keccak256 calculates and returns the Keccak256 hash of the input data.
func keccak256(data ...[]byte) []byte {
	d := sha3.NewKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

//生成添加校验码的地址
func addrByteToString(addrByte []byte) (addrStr string) {
	unchecksummed := hex.EncodeToString(addrByte)
	sha := sha3.NewKeccak256()
	sha.Write([]byte(unchecksummed))
	hash := sha.Sum(nil)

	result := []byte(unchecksummed)
	for i := 0; i < len(result); i++ {
		hashByte := hash[i/2]
		if i%2 == 0 {
			hashByte = hashByte >> 4
		} else {
			hashByte &= 0xf
		}
		if result[i] > '9' && hashByte > 7 {
			result[i] -= 32
		}
	}
	return "0x" + string(result)
}

//私钥生成公钥(压缩形式)
func (t EthBaseTransformer) PrivKeyToPub(priv []byte) (pub []byte, err error) {
	if len(priv) != 32 {
		return nil, fmt.Errorf("invalid priv key byte")
	}
	// pub = secp256k1.UncompressedPubkeyFromSeckey(priv)
	pub = secp256k1.PubkeyFromSeckey(priv)
	return
}

// 公钥生成地址(带校验码)
// 支持压缩和非压缩形式公钥，压缩形式会先转换成非压缩形式再生成地址，所以生成的地址是相同的
func (t EthBaseTransformer) PubKeyToAddress(pub []byte) (addr string, err error) {
	if len(pub) != 33 && len(pub) != 65 {
		return "", fmt.Errorf("invalid priv key byte")
	}

	//go-ethereum中地址是由非压缩形式公钥生成，所以这里如果传入的是压缩形式公钥，需要转换一下
	uncompressPub := pub
	if len(pub) == 33 {
		uncompressPub = secp256k1.UncompressPubkey(pub)
	}
	addrByte := keccak256(uncompressPub[1:])[12:]
	addr = addrByteToString(addrByte)
	return
}
