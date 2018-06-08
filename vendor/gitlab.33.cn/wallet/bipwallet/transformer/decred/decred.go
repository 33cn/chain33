//实现DCR币种的公钥和地址生成
package decred

import (
	"github.com/dchest/blake256"
	"github.com/haltingstate/secp256k1-go"
	"github.com/mr-tron/base58/base58"
	"golang.org/x/crypto/ripemd160"

	"fmt"
)

type DcrTransformer struct {
	//版本号前缀（表明网络类型和地址类型）
	prefix []byte
}

//checksum: first four bytes of double-BLAKE256.
func checksum(input []byte) (cksum [4]byte) {
	h := blake256.New()
	h.Write(input)
	intermediateHash := h.Sum(nil)
	h.Reset()
	h.Write(intermediateHash)
	finalHash := h.Sum(nil)
	copy(cksum[:], finalHash[:])
	return
}

//将uncompressed和hybrid形式的公钥转换成compressed形式
func toCompressed(input []byte) []byte {
	if len(input) != 65 {
		return nil
	}
	b := make([]byte, 0, 33)
	var format byte = 0x02
	//判断公钥的y参数是否为奇数
	if input[64]&0x1 == 1 {
		format = 0x03
	}
	b = append(b, format)
	b = append(b, input[1:33]...)
	return b
}

//TODO: 支持Decred中的另外两种加密算法(ed25519和SecSchnorr)
//32字节私钥生成压缩格式公钥(使用secp256k1算法)
func (t DcrTransformer) PrivKeyToPub(priv []byte) (pub []byte, err error) {
	if len(priv) != 32 {
		return nil, fmt.Errorf("invalid priv key byte")
	}
	pub = secp256k1.PubkeyFromSeckey(priv)
	// uncompressPub := secp256k1.UncompressPubkey(pub)
	return
}

//TODO: 支持不同的地址类型(P2PK)
//公钥(压缩、非压缩和混合形式)生成P2PKH类型的地址(以"Ds"打头)
func (t DcrTransformer) PubKeyToAddress(pub []byte) (addr string, err error) {
	//压缩格式 与 非压缩格式
	if len(pub) != 33 && len(pub) != 65 {
		return "", fmt.Errorf("invalid public key byte")
	}
	//使用压缩格式公钥生成地址
	compressPub := pub
	if len(pub) == 65 {
		compressPub = toCompressed(pub)
	}
	//160hash
	h := blake256.New()
	h.Write(compressPub)
	ripemd160h := ripemd160.New()
	ripemd160h.Write(h.Sum(nil))
	//添加版本号
	hash160res := append(t.prefix, ripemd160h.Sum([]byte(""))...)
	//添加校验码
	cksum := checksum(hash160res)
	address := append(hash160res, cksum[:]...)
	//地址进行base58编码
	addr = base58.Encode(address)
	return
}
