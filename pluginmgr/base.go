package pluginmgr

import (
	"github.com/spf13/cobra"
)

type PluginBase struct {
	Name     string
	ExecName string
	RPC      func(name string, s RPCServer)
	Exec     func(name string)
	Cmd      func() *cobra.Command
}

func (p *PluginBase) GetName() string {
	return p.Name
}

func (p *PluginBase) GetExecutorName() string {
	return p.ExecName
}

func (p *PluginBase) InitExec() {
	p.Exec(p.ExecName)
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

func (p *PluginBase) AddRPC(c RPCServer) {
	if p.RPC != nil {
		p.RPC(p.GetExecutorName(), c)
	}
}
