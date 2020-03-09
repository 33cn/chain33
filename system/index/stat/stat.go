// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stat

import (
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/index"
	"github.com/33cn/chain33/types"
)

var (
	name = "stat"
	elog = log.New("module", "system/index/stat")
)

func init() {
	plugin.RegisterPlugin("stat", newStat)
}

type statPlugin struct {
	*plugin.Base
	*plugin.Flag
}

func newStat() plugin.Plugin {
	p := &statPlugin{
		Base: &plugin.Base{},
		Flag: &plugin.Flag{},
	}
	p.SetName(name)
	return p
}

func (p *statPlugin) CheckEnable(enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	kvs, ok, err = p.CheckFlag(p, plugin.FlagKey(name), enable)
	if err == types.ErrDBFlag {
		panic("stat config is enable, it must be synchronized from 0 height ")
	}
	return kvs, ok, err
}

func (p *statPlugin) ExecLocal(data *types.BlockDetail) ([]*types.KeyValue, error) {
	return countInfo(data)
}

func (p *statPlugin) ExecDelLocal(data *types.BlockDetail) ([]*types.KeyValue, error) {
	return delCountInfo(data)
}

func countInfo(b *types.BlockDetail) ([]*types.KeyValue, error) {
	var kvset types.LocalDBSet
	//保存挖矿统计数据
	ticketkv, err := countTicket(b)
	if err != nil {
		return nil, err
	}
	if ticketkv == nil {
		return nil, nil
	}
	kvset.KV = ticketkv.KV
	return kvset.KV, nil
}

func delCountInfo(b *types.BlockDetail) ([]*types.KeyValue, error) {
	var kvset types.LocalDBSet
	//删除挖矿统计数据
	ticketkv, err := delCountTicket(b)
	if err != nil {
		return nil, err
	}
	if ticketkv == nil {
		return nil, nil
	}
	kvset.KV = ticketkv.KV
	return kvset.KV, nil
}

//这两个功能需要重构到 ticket 里面去。
//有些功能需要开启选项，才会启用功能。并且功能必须从0开始
func countTicket(b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}

func delCountTicket(b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}
