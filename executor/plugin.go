// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"sync"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

//plugin 主要用于处理 execlocal 和 execdellocal 时候的全局kv的处理
//每个插件都有插件是否开启这个插件的判断，如果不开启，执行的时候会被忽略

type plugin interface {
	CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error)
	ExecLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error)
	ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error)
}

var globalPlugins = make(map[string]plugin)

// RegisterPlugin register plugin
func RegisterPlugin(name string, p plugin) {
	if _, ok := globalPlugins[name]; ok {
		panic("plugin exist " + name)
	}
	globalPlugins[name] = p
}

type pluginBase struct {
	flag int64
	mu   sync.Mutex
}

func (base *pluginBase) checkFlag(executor *executor, flagKey []byte, enable bool) (kvset []*types.KeyValue, ok bool, err error) {
	if !enable {
		return nil, false, nil
	}
	base.mu.Lock()
	defer base.mu.Unlock()
	if base.flag == 0 {
		flag, err := loadFlag(executor.localDB, flagKey)
		if err != nil {
			return nil, false, err
		}
		base.flag = flag
	}
	if executor.height != 0 && base.flag == 0 {
		return nil, false, types.ErrDBFlag
	}
	if executor.height == 0 {
		base.flag = 1
		kvset = append(kvset, types.FlagKV(flagKey, base.flag))
	}
	return kvset, true, nil
}

func loadFlag(localDB dbm.KVDB, key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := localDB.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound {
		return 0, nil
	}
	return 0, err
}
