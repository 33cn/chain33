// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package store store the world - state data
package store

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// Storecreate store queue module
type Storecreate func(cfg *types.Store, sub []byte) queue.Module

var regStore = make(map[string]Storecreate)

// Reg 注册 store driver
func Reg(name string, create Storecreate) {
	if create == nil {
		panic("Store: Register driver is nil")
	}
	if _, dup := regStore[name]; dup {
		panic("Store: Register called twice for driver " + name)
	}
	regStore[name] = create
}

// Load load StoreCreate by name
func Load(name string) (create Storecreate, err error) {
	if driver, ok := regStore[name]; ok {
		return driver, nil
	}
	return nil, types.ErrNotFound
}
