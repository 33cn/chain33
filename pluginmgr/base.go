// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package pluginmgr 插件管理模块，负责插件初始化等功能
package pluginmgr

import (
	"github.com/33cn/chain33/rpc/types"
	wcom "github.com/33cn/chain33/wallet/common"
	"github.com/spf13/cobra"
)

// PluginBase plugin module base struct
type PluginBase struct {
	Name     string
	ExecName string
	RPC      func(name string, s types.RPCServer)
	Exec     func(name string, sub []byte)
	Wallet   func(walletBiz wcom.WalletOperate, sub []byte)
	Cmd      func() *cobra.Command
}

// GetName 获取整个插件的包名，用以计算唯一值、做前缀等
func (p *PluginBase) GetName() string {
	return p.Name
}

// GetExecutorName 获取插件中执行器名
func (p *PluginBase) GetExecutorName() string {
	return p.ExecName
}

// InitExec init exec
func (p *PluginBase) InitExec(sub map[string][]byte) {
	subcfg, ok := sub[p.ExecName]
	if !ok {
		subcfg = nil
	}
	p.Exec(p.ExecName, subcfg)
}

// InitWallet init wallet plugin
func (p *PluginBase) InitWallet(walletBiz wcom.WalletOperate, sub map[string][]byte) {
	subcfg, ok := sub[p.ExecName]
	if !ok {
		subcfg = nil
	}
	p.Wallet(walletBiz, subcfg)
}

// AddCmd add Command for plugin cli
func (p *PluginBase) AddCmd(rootCmd *cobra.Command) {
	if p.Cmd != nil {
		cmd := p.Cmd()
		if cmd == nil {
			return
		}
		rootCmd.AddCommand(cmd)
	}
}

// AddRPC add Rpc for plugin
func (p *PluginBase) AddRPC(c types.RPCServer) {
	if p.RPC != nil {
		p.RPC(p.GetExecutorName(), c)
	}
}
