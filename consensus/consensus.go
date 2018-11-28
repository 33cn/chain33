// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package consensus 共识相关的模块
package consensus

import (
	"reflect"

	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/system/consensus"
	"github.com/33cn/chain33/types"
)

// New new consensus queue module
func New(cfg *types.Consensus, sub map[string][]byte) queue.Module {
	con, err := consensus.Load(cfg.Name)
	if err != nil {
		panic("Unsupported consensus type:" + cfg.Name + " " + err.Error())
	}
	subcfg, ok := sub[cfg.Name]
	if !ok {
		subcfg = nil
	}
	obj := con(cfg, subcfg)
	consensus.QueryData.SetThis(cfg.Name, reflect.ValueOf(obj))
	return obj
}
