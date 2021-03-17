// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto

//Crypto 加密
type Crypto interface {
	GenKey() (PrivKey, error)
	SignatureFromBytes([]byte) (Signature, error)
	PrivKeyFromBytes([]byte) (PrivKey, error)
	PubKeyFromBytes([]byte) (PubKey, error)
	Validate(msg, pub, sig []byte) error
}

//AggregateCrypto 聚合签名
type AggregateCrypto interface {
	Aggregate(sigs []Signature) (Signature, error)
	AggregatePublic(pubs []PubKey) (PubKey, error)
	VerifyAggregatedOne(pubs []PubKey, m []byte, sig Signature) error
	VerifyAggregatedN(pubs []PubKey, ms [][]byte, sig Signature) error
}

//PrivKey 私钥
type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) Signature
	PubKey() PubKey
	Equals(PrivKey) bool
}

//Signature 签名
type Signature interface {
	Bytes() []byte
	IsZero() bool
	String() string
	Equals(Signature) bool
}

//PubKey 公钥
type PubKey interface {
	Bytes() []byte
	KeyString() string
	VerifyBytes(msg []byte, sig Signature) bool
	Equals(PubKey) bool
}

// DriverInitFn 加密插件的初始化接口
type DriverInitFn func(subCfg []byte)

//CertSignature 签名
type CertSignature struct {
	Signature []byte
	Cert      []byte
}

// Config crypto模块配置
type Config struct {
	//支持指定若干签名类型，不配置默认除none以外的其他签名类型均启用
	Types []string `json:"types,omitempty"`
}
