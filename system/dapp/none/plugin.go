// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package none 系统级dapp，执行内容为空
package none

import (
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/system/dapp/none/executor"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     "none",
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      nil,
		RPC:      nil,
	})
}
