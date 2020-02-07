// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/types"
)

//plugin 主要用于处理 execlocal 和 execdellocal 时候的全局kv的处理
//每个插件都有插件是否开启这个插件的判断，如果不开启，执行的时候会被忽略

// Plugin executor plugin
type Plugin interface {
	CheckEnable(executor *executor, enable bool) (kvs []*types.KeyValue, ok bool, err error)
	ExecLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error)
	ExecDelLocal(executor *executor, data *types.BlockDetail) ([]*types.KeyValue, error)
}

var globalPlugins = make(map[string]Plugin)

// RegisterPlugin register plugin
func RegisterPlugin(name string, p Plugin) {
	if _, ok := globalPlugins[name]; ok {
		panic("plugin exist " + name)
	}
	globalPlugins[name] = p
}

// PluginBase plugin base for impl plugin
