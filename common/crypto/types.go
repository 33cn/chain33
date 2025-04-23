// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto

// Crypto 加密
type Crypto interface {
	GenKey() (PrivKey, error)
	SignatureFromBytes([]byte) (Signature, error)
	PrivKeyFromBytes([]byte) (PrivKey, error)
	PubKeyFromBytes([]byte) (PubKey, error)
	Validate(msg, pub, sig []byte) error
}

// AggregateCrypto 聚合签名
type AggregateCrypto interface {
	Aggregate(sigs []Signature) (Signature, error)
	AggregatePublic(pubs []PubKey) (PubKey, error)
	VerifyAggregatedOne(pubs []PubKey, m []byte, sig Signature) error
	VerifyAggregatedN(pubs []PubKey, ms [][]byte, sig Signature) error
}

// PrivKey 私钥
type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) Signature
	PubKey() PubKey
	Equals(PrivKey) bool
}

// Signature 签名
type Signature interface {
	Bytes() []byte
	IsZero() bool
	String() string
	Equals(Signature) bool
}

// PubKey 公钥
type PubKey interface {
	Bytes() []byte
	KeyString() string
	VerifyBytes(msg []byte, sig Signature) bool
	Equals(PubKey) bool
}

// CertSignature 签名
type CertSignature struct {
	Signature []byte
	Cert      []byte
}

// Config crypto模块配置
type Config struct {
	//支持只指定若干加密类型，不配置默认启用所有的加密插件, 如 types=["secp256k1", "sm2"]
	EnableTypes []string `json:"enableTypes,omitempty"`
	//支持对EnableTypes的每个加密插件分别设置启用高度, 不配置采用内置的启用高度
	// [crypto.enableHeight]
	// secp256k1=0
	EnableHeight map[string]int64 `json:"enableHeight,omitempty"`
}

// DriverInitFunc 插件初始化接口，参数是序列化的json数据，需要unmarshal为自定义的结构
type DriverInitFunc func(jsonCfg []byte)

// Driver 加密插件及相关信息
type Driver struct {
	name         string
	crypto       Crypto
	initFunc     DriverInitFunc
	isCGO        bool  // 是否为cgo编译
	enable       bool  // 是否启用
	enableHeight int64 // 启用高度
	typeID       int32 //类型值
}

// RegOption  Register Driver可选参数，设置相关参数默认值
type RegOption func(*Driver) error

// LoadOption load crypto可选参数
type LoadOption func(*Driver) error
