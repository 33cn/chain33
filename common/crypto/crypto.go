// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package crypto 加解密、签名接口定义
package crypto

import (
	"errors"
	"fmt"
	"sync"
)

//ErrNotSupportAggr 不支持聚合签名
var ErrNotSupportAggr = errors.New("AggregateCrypto not support")

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

//Crypto 加密
type Crypto interface {
	GenKey() (PrivKey, error)
	SignatureFromBytes([]byte) (Signature, error)
	PrivKeyFromBytes([]byte) (PrivKey, error)
	PubKeyFromBytes([]byte) (PubKey, error)
}

//AggregateCrypto 聚合签名
type AggregateCrypto interface {
	Aggregate(sigs []Signature) (Signature, error)
	AggregatePublic(pubs []PubKey) (PubKey, error)
	VerifyAggregatedOne(pubs []PubKey, m []byte, sig Signature) error
	VerifyAggregatedN(pubs []PubKey, ms [][]byte, sig Signature) error
}

//ToAggregate 判断签名是否可以支持聚合签名，并且返回聚合签名的接口
func ToAggregate(c Crypto) (AggregateCrypto, error) {
	if aggr, ok := c.(AggregateCrypto); ok {
		return aggr, nil
	}
	return nil, ErrNotSupportAggr
}

var (
	drivers     = make(map[string]Crypto)
	driversCGO  = make(map[string]Crypto)
	driversType = make(map[string]int)
	driverMutex sync.Mutex
)

//Register 注册加密算法，允许同种加密算法的cgo版本同时注册
func Register(name string, driver Crypto, isCGO bool) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	d := drivers
	if isCGO {
		d = driversCGO
	}
	if _, dup := d[name]; dup {
		panic("crypto: Register called twice for driver " + name)
	}
	d[name] = driver
}

//RegisterType 注册类型
func RegisterType(name string, ty int) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	for n, t := range driversType {
		//由于可能存在cgo版本，允许name，ty都相等情况，重复注册
		//或者都不相等，添加新的类型
		//不允许只有一个相等的情况，即不允许修改已经存在的ty或者name
		if (n == name && t != ty) || (n != name && t == ty) {
			panic(fmt.Sprintf("crypto: Register Conflict, exist=(%s,%d), register=(%s, %d)", n, t, name, ty))
		}
	}
	driversType[name] = ty
}

//GetName 获取name
func GetName(ty int) string {
	for name, t := range driversType {
		if t == ty {
			return name
		}
	}
	return "unknown"
}

//GetType 获取type
func GetType(name string) int {
	if ty, ok := driversType[name]; ok {
		return ty
	}
	return 0
}

//New new
func New(name string) (c Crypto, err error) {

	//优先使用性能更好的cgo版本
	c, ok := driversCGO[name]
	if ok {
		return c, nil
	}
	//不存在cgo, 加载普通版本
	c, ok = drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
	}
	return c, err
}

//CertSignature 签名
type CertSignature struct {
	Signature []byte
	Cert      []byte
}
