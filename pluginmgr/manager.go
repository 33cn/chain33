// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pluginmgr

import (
	"sync"

	"github.com/33cn/chain33/rpc/types"
	wcom "github.com/33cn/chain33/wallet/common"
	"github.com/spf13/cobra"
)

var pluginItems = make(map[string]Plugin)

var once = &sync.Once{}

// InitExec init exec
func InitExec(sub map[string][]byte) {
	once.Do(func() {
		for _, item := range pluginItems {
			item.InitExec(sub)
		}
	})
}

// InitWallet init wallet plugin
func InitWallet(wallet wcom.WalletOperate, sub map[string][]byte) {
	once.Do(func() {
		for _, item := range pluginItems {
			item.InitWallet(wallet, sub)
		}
	})
}

// HasExec check is have the name exec
func HasExec(name string) bool {
	for _, item := range pluginItems {
		if item.GetExecutorName() == name {
			return true
		}
	}
	return false
}

// Register Register plugin
func Register(p Plugin) {
	if p == nil {
		panic("plugin param is nil")
	}
	packageName := p.GetName()
	if len(packageName) == 0 {
		panic("plugin package name is empty")
	}
	if _, ok := pluginItems[packageName]; ok {
		panic("execute plugin item is existed. name = " + packageName)
	}
	pluginItems[packageName] = p
}

// AddCmd add Command for plugin
func AddCmd(rootCmd *cobra.Command) {
	for _, item := range pluginItems {
		item.AddCmd(rootCmd)
	}
}

// AddRPC add Rpc
func AddRPC(s types.RPCServer) {
	for _, item := range pluginItems {
		item.AddRPC(s)
	}
}
