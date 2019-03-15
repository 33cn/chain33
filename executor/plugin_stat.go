// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/types"
)

func init() {
	RegisterPlugin("stat", &statPlugin{})
}

type statPlugin struct {
	pluginBase
}

func (p *statPlugin) CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error) {
	kvs, ok, err = p.checkFlag(executor, types.StatisticFlag(), enable)
	if err == types.ErrDBFlag {
		panic("stat config is enable, it must be synchronized from 0 height ")
	}
	return kvs, ok, err
}

func (p *statPlugin) ExecLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	return countInfo(executor, data)
}

func (p *statPlugin) ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error) {
	return delCountInfo(executor, data)
}

func countInfo(ex *executor, b *types.BlockDetail) ([]*types.KeyValue, error) {
	var kvset types.LocalDBSet
	//保存挖矿统计数据
	ticketkv, err := countTicket(ex, b)
	if err != nil {
		return nil, err
	}
	if ticketkv == nil {
		return nil, nil
	}
	kvset.KV = ticketkv.KV
	return kvset.KV, nil
}

func delCountInfo(ex *executor, b *types.BlockDetail) ([]*types.KeyValue, error) {
	var kvset types.LocalDBSet
	//删除挖矿统计数据
	ticketkv, err := delCountTicket(ex, b)
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
func countTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}

func delCountTicket(ex *executor, b *types.BlockDetail) (*types.LocalDBSet, error) {
	return nil, nil
}
