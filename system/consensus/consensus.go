// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package consensus

import (
	"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
)

type ConsensusCreate func(cfg *types.Consensus, sub []byte) queue.Module

var regConsensus = make(map[string]ConsensusCreate)
var QueryData = types.NewQueryData("Query_")

func Reg(name string, create ConsensusCreate) {
	if create == nil {
		panic("Consensus: Register driver is nil")
	}
	if _, dup := regConsensus[name]; dup {
		panic("Consensus: Register called twice for driver " + name)
	}
	regConsensus[name] = create
}

func Load(name string) (create ConsensusCreate, err error) {
	if driver, ok := regConsensus[name]; ok {
		return driver, nil
	}
	return nil, types.ErrNotFound
}
