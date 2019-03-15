// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package btcbase 转换基于比特币地址规则的币种
//使用此规则的币种有：BTC、BCH、LTC、ZEC、USDT、 BTY
package btcbase

import (
	"crypto/sha256"

	"github.com/haltingstate/secp256k1-go"
	"github.com/mr-tron/base58/base58"
	"golang.org/x/crypto/ripemd160"

	"fmt"
)

// btcBaseTransformer 转换基于比特币地址规则的币种实现类
type btcBaseTransformer struct {
	prefix []byte //版本号前缀
}

// Base58ToByte base58转字节形式
func (t btcBaseTransformer) Base58ToByte(str string) (bin []byte, err error) {
	bin, err = base58.Decode(str)
	return
}

// ByteToBase58 字节形式转base58编码
func (t btcBaseTransformer) ByteToBase58(bin []byte) (str string) {
	str = base58.Encode(bin)
	return
}

//TODO: 根据私钥类型进行判断，选择输出压缩或非压缩公钥

// PrivKeyToPub 32字节私钥生成压缩格式公钥
func (t btcBaseTransformer) PrivKeyToPub(priv []byte) (pub []byte, err error) {
	if len(priv) != 32 {
		return nil, fmt.Errorf("invalid priv key byte")
	}
	pub = secp256k1.PubkeyFromSeckey(priv)
	// uncompressPub := secp256k1.UncompressPubkey(pub)
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

// PubKeyToAddress 传入压缩或非压缩形式的公钥，生成base58编码的地址
//（压缩和非压缩形式的公钥生成的地址是不同的，但都是合法的）
func (t btcBaseTransformer) PubKeyToAddress(pub []byte) (addr string, err error) {
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
