// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package crypto 加解密、签名接口定义
package crypto

import (
	"fmt"
	"sync"
)

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

var (
	drivers     = make(map[string]Crypto)
	driversType = make(map[string]int)
)

var driverMutex sync.Mutex

//const
const (
	SignNameED25519 = "ed25519"
)

//Register 注册
func Register(name string, driver Crypto) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	if driver == nil {
		panic("crypto: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("crypto: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

//RegisterType 注册类型
func RegisterType(name string, ty int) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	if _, dup := driversType[name]; dup {
		panic("crypto: Register(ty) called twice for driver " + name)
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
	driverMutex.Lock()
	defer driverMutex.Unlock()
	c, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
		return
	}

	return c, nil
}

//CertSignature 签名
type CertSignature struct {
	Signature []byte
	Cert      []byte
}
