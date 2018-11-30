// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package transformer 通过私钥生成所选币种的公钥和地址
package transformer

import (
	"fmt"
	"sync"
)

// Transformer 过私钥生成所选币种的公钥和地址
type Transformer interface {
	PrivKeyToPub(priv []byte) (pub []byte, err error)
	PubKeyToAddress(pub []byte) (add string, err error)
}

var (
	driversMu sync.RWMutex
	drivers   = make(map[string]Transformer)
)

// Register 对不同币种的Transformer进行注册
func Register(name string, driver Transformer) {
	driversMu.Lock()
	defer driversMu.Unlock()
	if driver == nil {
		panic("transformer: Register driver is nil")
	}
	if _, dup := drivers[name]; dup {
		panic("transformer: Register called twice for driver " + name)
	}
	drivers[name] = driver
}

// New 提供币种名称返回相应的Transformer对象
func New(name string) (t Transformer, err error) {
	driversMu.RLock()
	defer driversMu.RUnlock()
	t, ok := drivers[name]
	if !ok {
		err = fmt.Errorf("unknown Transformer %q", name)
		return
	}

	return t, nil
}
