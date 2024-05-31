// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package consensus 系统基础共识包
package consensus

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

// Create 创建共识
type Create func(cfg *types.Consensus, sub []byte) queue.Module

var regConsensus = make(map[string]Create)

// QueryData 检索数据
var QueryData = types.NewQueryData("Query_")

// Reg ...
func Reg(name string, create Create) {
	if create == nil {
		panic("Consensus: Register driver is nil")
	}
	if _, dup := regConsensus[name]; dup {
		panic("Consensus: Register called twice for driver " + name)
	}
	regConsensus[name] = create
}

// Load 加载
func Load(name string) (create Create, err error) {
	if driver, ok := regConsensus[name]; ok {
		return driver, nil
	}
	return nil, types.ErrNotFound
}

// Committer state commiter
type Committer interface {
	Init(base *BaseClient, subCfg []byte)
	SubMsg(msg *queue.Message)
}

var committers = make(map[string]Committer)

// RegCommitter register committer
func RegCommitter(name string, c Committer) {

	if c == nil {
		panic("RegCommitter: committer is nil")
	}
	if _, dup := committers[name]; dup {
		panic("RegCommitter: duplicate committer " + name)
	}
	committers[name] = c
}

// LoadCommiter load
func LoadCommiter(name string) Committer {

	return committers[name]
}

// Context 共识相关依赖
type Context struct {
	Base *BaseClient
}
