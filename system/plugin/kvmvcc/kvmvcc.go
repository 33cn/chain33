// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
)

func init() {
	plugin.RegisterPlugin("mvcc", &mvccPlugin{})
}

type mvccPlugin struct {
	plugin.Base
	plugin.Flag
}

func (p *mvccPlugin) CheckEnable(enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	kvs, ok, err = p.CheckFlag(p, types.FlagKeyMVCC, enable)
	if err == types.ErrDBFlag {
		panic("mvcc config is enable, it must be synchronized from 0 height ")
	}
	return kvs, ok, err
}

func (p *mvccPlugin) ExecLocal(data *types.BlockDetail) (kvs []*types.KeyValue, err error) {
	kvs = AddMVCC(p.GetLocalDB(), data)
	return kvs, nil
}

func (p *mvccPlugin) ExecDelLocal(data *types.BlockDetail) ([]*types.KeyValue, error) {
	kvs := DelMVCC(p.GetLocalDB(), data)
	return kvs, nil
}

// AddMVCC convert key value to mvcc kv data
func AddMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	kvs := detail.KV
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	//检查版本号是否是连续的
	kvlist, err := mvcc.AddMVCC(kvs, hash, detail.PrevStatusHash, detail.Block.Height)
	if err != nil {
		panic(err)
	}
	return kvlist
}

// DelMVCC convert key value to mvcc kv data
func DelMVCC(db dbm.KVDB, detail *types.BlockDetail) (kvlist []*types.KeyValue) {
	hash := detail.Block.StateHash
	mvcc := dbm.NewSimpleMVCC(db)
	kvlist, err := mvcc.DelMVCC(hash, detail.Block.Height, true)
	if err != nil {
		panic(err)
	}
	return kvlist
}
