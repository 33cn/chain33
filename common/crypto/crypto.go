// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto

import (
	"fmt"
	"sync"
)

type PrivKey interface {
	Bytes() []byte
	Sign(msg []byte) Signature
	PubKey() PubKey
	Equals(PrivKey) bool
}

type Signature interface {
	Bytes() []byte
	IsZero() bool
	String() string
	Equals(Signature) bool
}

type PubKey interface {
	Bytes() []byte
	KeyString() string
	VerifyBytes(msg []byte, sig Signature) bool
	Equals(PubKey) bool
}

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

const (
	SignNameED25519 = "ed25519"
)

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

func RegisterType(name string, ty int) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	if _, dup := driversType[name]; dup {
		panic("crypto: Register(ty) called twice for driver " + name)
	}
	driversType[name] = ty
}

func GetName(ty int) string {
	for name, t := range driversType {
		if t == ty {
			return name
		}
	}
	return "unknown"
}

func GetType(name string) int {
	if ty, ok := driversType[name]; ok {
		return ty
	}
	return 0
}

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

type CertSignature struct {
	Signature []byte
	Cert      []byte
}
