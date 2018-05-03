# decred

本模块实现Decred中公钥和地址的生成

## 涉及币种
- DCR

## 依赖清单
- github.com/dchest/blake256
- github.com/mr-tron/base58/base58
- github.com/haltingstate/secp256k1-go
- golang.org/x/crypto/ripemd160

## 输入输出格式说明

```golang
//32字节私钥生成压缩格式公钥(使用secp256k1算法)
func (t DcrTransformer) PrivKeyToPub(priv []byte) (pub []byte, err error)
```
```golang
//公钥(压缩、非压缩和混合形式)生成P2PKH类型的地址(以"Ds"打头)
func (t DcrTransformer) PubKeyToAddress(pub []byte) (addr string, err error)
```

## 生成规则

Decred支持三种加密算法：
- Secp256k1
- Ed25519
- SecSchnorr

而且有三种类型的地址：
- pay-to-pubkey(P2PK) 
- pay-to-pubkey-hash(P2PKH)
- pay-to-script-hash(P2PH)

再加上不同的网络参数，Decred中有非常多种地址前缀：
```golang
// Address encoding magics
NetworkAddressPrefix: "T",
PubKeyAddrID:         [2]byte{0x28, 0xf7}, // starts with Tk
PubKeyHashAddrID:     [2]byte{0x0f, 0x21}, // starts with Ts
PKHEdwardsAddrID:     [2]byte{0x0f, 0x01}, // starts with Te
PKHSchnorrAddrID:     [2]byte{0x0e, 0xe3}, // starts with TS
ScriptHashAddrID:     [2]byte{0x0e, 0xfc}, // starts with Tc
PrivateKeyID:         [2]byte{0x23, 0x0e}, // starts with Pt

// Address encoding magics
NetworkAddressPrefix: "D",
PubKeyAddrID:         [2]byte{0x13, 0x86}, // starts with Dk
PubKeyHashAddrID:     [2]byte{0x07, 0x3f}, // starts with Ds
PKHEdwardsAddrID:     [2]byte{0x07, 0x1f}, // starts with De
PKHSchnorrAddrID:     [2]byte{0x07, 0x01}, // starts with DS
ScriptHashAddrID:     [2]byte{0x07, 0x1a}, // starts with Dc
PrivateKeyID:         [2]byte{0x22, 0xde}, // starts with Pm
```

本模块实现的是Ds前缀的地址(使用spec256k1加密，P2PKH类型的地址)，也是最常见的一种。

私钥生成公钥的部分和比特币是一样的，使用的都是spec256k1算法。

公钥生成地址的部分在流程上也是和比特币一样的，但是有一处不同：Decred所使用的哈希算法是blake256，而比特币用的是sha256。
