package pluginmgr

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/rpc/types"
)

type PluginBase struct {
	Name     string
	ExecName string
	RPC      func(name string, s types.RPCServer)
	Exec     func(name string, sub []byte)
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
