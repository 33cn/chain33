// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pluginmgr

import (
	"github.com/33cn/chain33/rpc/types"
	wcom "github.com/33cn/chain33/wallet/common"
	"github.com/spf13/cobra"
)

type PluginBase struct {
	Name     string
	ExecName string
	RPC      func(name string, s types.RPCServer)
	Exec     func(name string, sub []byte)
	Wallet   func(walletBiz wcom.WalletOperate, sub []byte)
	Cmd      func() *cobra.Command
}

func (p *PluginBase) GetName() string {
	return p.Name
}

func (p *PluginBase) GetExecutorName() string {
	return p.ExecName
}

func (p *PluginBase) InitExec(sub map[string][]byte) {
	subcfg, ok := sub[p.ExecName]
	if !ok {
		subcfg = nil
	}
	p.Exec(p.ExecName, subcfg)
}

func (p *PluginBase) InitWallet(walletBiz wcom.WalletOperate, sub map[string][]byte) {
	subcfg, ok := sub[p.ExecName]
	if !ok {
		subcfg = nil
	}
	p.Wallet(walletBiz, subcfg)
}

func (p *PluginBase) AddCmd(rootCmd *cobra.Command) {
	if p.Cmd != nil {
		cmd := p.Cmd()
		if cmd == nil {
			return
		}
		rootCmd.AddCommand(cmd)
	}
}

func (p *PluginBase) AddRPC(c types.RPCServer) {
	if p.RPC != nil {
		p.RPC(p.GetExecutorName(), c)
	}
}
