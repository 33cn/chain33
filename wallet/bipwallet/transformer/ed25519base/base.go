package ed25519base

import (
	"crypto/sha256"
	"fmt"
	bip39 "github.com/33cn/chain33/wallet/bipwallet/go-bip39"
	"github.com/NebulousLabs/Sia/crypto"
	"github.com/mr-tron/base58/base58"
	"golang.org/x/crypto/ripemd160"
)

type baseTransformer struct {
	prefix []byte //版本号前缀
}

func (t baseTransformer) PrivKeyToPub(priv []byte) ([]byte, error) {
	if len(priv) != 64 {
		return nil, fmt.Errorf("invalid priv key byte")
	}
	//将私钥转为Sia内置的SecretKey类型
	var sk crypto.SecretKey
	copy(sk[:], priv)
	//取私钥的后32字节即为公钥
	pk := sk.PublicKey()

	var raw []byte = make([]byte, 33)
	raw[0] = 0x03
	copy(raw[1:], pk[:])
	return raw, nil
}

func (t baseTransformer) PubKeyToAddress(pub []byte) (addr string, err error) {

	if len(pub) != 33 && len(pub) != 65 { //压缩格式 与 非压缩格式
		return "", fmt.Errorf("invalid public key byte")
	}

	sha256h := sha256.New()
	_, err = sha256h.Write(pub)
	if err != nil {
		return "", err
	}
	//160hash
	ripemd160h := ripemd160.New()
	_, err = ripemd160h.Write(sha256h.Sum([]byte("")))
	if err != nil {
		return "", err
	}
	//添加版本号
	hash160res := append(t.prefix, ripemd160h.Sum([]byte(""))...)

	//添加校验码
	cksum := checksum(hash160res)
	address := append(hash160res, cksum[:]...)

	//地址进行base58编码
	addr = base58.Encode(address)
	return
}

//checksum: first four bytes of double-SHA256.
func checksum(input []byte) (cksum [4]byte) {
	h := sha256.New()
	_, err := h.Write(input)
	if err != nil {
		return
	}
	intermediateHash := h.Sum(nil)
	h.Reset()
	_, err = h.Write(intermediateHash)
	if err != nil {
		return
	}
	finalHash := h.Sum(nil)
	copy(cksum[:], finalHash[:])
	return
}

//通过助记词形式的seed生成私钥和公钥,一个seed根据不同的index可以生成许多组密钥
func MnemonicToSecretkey(mnemonic string, index uint64) (secretKey, publicKey string, err error) {
	//字符串形式的助记词(英语单词)转成字节形式的seed
	seed, err := bip39.NewSeedWithErrorChecking(mnemonic, "")
	if err != nil {
		return "", "", err
	}

	sk, pk := crypto.GenerateKeyPairDeterministic(crypto.HashAll(seed, index))
	secretKey = fmt.Sprintf("%x", sk)
	var raw []byte = make([]byte, 33)
	raw[0] = 0x03
	copy(raw[1:], pk[:])
	publicKey = fmt.Sprintf("%x", raw)
	return
}
