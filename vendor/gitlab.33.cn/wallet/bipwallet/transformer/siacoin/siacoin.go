//转换SC(云储币)
package siacoin

import (
	"fmt"
	"github.com/NebulousLabs/Sia/crypto"
	"github.com/NebulousLabs/Sia/modules"
	"github.com/NebulousLabs/Sia/types"
)

type ScTransformer struct{}

//64字节私钥生成32字节公钥
func (t ScTransformer) PrivKeyToPub(priv []byte) (pub []byte, err error) {
	if len(priv) != 64 {
		return nil, fmt.Errorf("invalid priv key byte")
	}
	//将私钥转为Sia内置的SecretKey类型
	var sk crypto.SecretKey
	copy(sk[:], priv)
	//取私钥的后32字节即为公钥
	pk := sk.PublicKey()
	pub = pk[:]
	return
}

//32字节公钥生成地址
func (t ScTransformer) PubKeyToAddress(pub []byte) (addr string, err error) {
	if len(pub) != 32 {
		return "", fmt.Errorf("invalid public key byte")
	}
	var pk crypto.PublicKey
	copy(pk[:], pub)

	//通过公钥构造UnlockConditions，再进行UnlockHash操作得到地址
	uc := types.UnlockConditions{
		PublicKeys:         []types.SiaPublicKey{types.Ed25519PublicKey(pk)},
		SignaturesRequired: 1,
	}
	address := uc.UnlockHash()
	addr = address.String()
	return
}

//通过助记词形式的seed生成私钥和公钥,一个seed根据不同的index可以生成许多组密钥
func SeedToSecretkey(seedStr string, index uint64) (secretKey, publicKey string, err error) {
	//字符串形式的助记词(英语单词)转成字节形式的seed
	seed, err := modules.StringToSeed(seedStr, "english")
	if err != nil {
		return "", "", err
	}
	sk, pk := crypto.GenerateKeyPairDeterministic(crypto.HashAll(seed, index))
	secretKey = fmt.Sprintf("%x", sk)
	publicKey = fmt.Sprintf("%x", pk)
	return
}
