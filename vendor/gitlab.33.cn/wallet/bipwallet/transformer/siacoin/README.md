# siacoin

本模块实现基于云储币规则的公钥和地址的生成

## 涉及币种
- SC

## 依赖清单
- github.com/NebulousLabs/Sia
- github.com/NebulousLabs/entropy-mnemonics
- github.com/NebulousLabs/fastrand
- github.com/NebulousLabs/merkletree
- golang.org/x/crypto/blake2b
- golang.org/x/text/unicode/norm
- github.com/NebulousLabs/bolt

## 输入输出格式说明

Sia使用的加密算法是Ed25519，输入的私钥为64字节，输出的公钥为32字节，地址是十六进制形式。

```golang
//64字节私钥生成32字节公钥
func (t ScTransformer) PrivKeyToPub(priv []byte) (pub []byte, err error)
```
```golang
//32字节公钥生成地址
func (t ScTransformer) PubKeyToAddress(pub []byte) (addr string, err error)
```

## 生成规则

Sia中的地址是通过seed生成的，每个wallet会有一个primary seed。  
一个primary seed 可以根据不同的index生成很多的地址，具体规则如下：

1. seed + index 进行hash，作为ed25519的随机种子，生成私钥+公钥

```
sk, pk := crypto.GenerateKeyPairDeterministic(crypto.HashAll(seed, index))
```

2. 私钥和公钥会组成spendablekey:

```
spendableKey{
	UnlockConditions: types.UnlockConditions{
		PublicKeys:         []types.SiaPublicKey{types.Ed25519PublicKey(pk)},
		SignaturesRequired: 1,
	},
	SecretKeys: []crypto.SecretKey{sk},
}
```

3. 地址是一种被称为UnlockHash的东西，通过UnlockConditions生成：

```
spendableKey.UnlockConditions.UnlockHash()
```

其中ed25519生成的私钥是64字节，公钥是32字节，私钥的后32字节就是公钥拼接上去的
