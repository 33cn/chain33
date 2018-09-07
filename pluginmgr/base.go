package pluginmgr

import (
	"github.com/spf13/cobra"
)

type PluginBase struct {
	Name     string
	ExecName string
	RPC      func(s RPCServer)
	Exec     func()
	Cmd      func() *cobra.Command
}

func (p *PluginBase) GetName() string {
	return p.Name
}

func (p *PluginBase) GetExecutorName() string {
	return p.ExecName
}

func (p *PluginBase) InitExec() {
	p.Exec()
}

func (p *PluginBase) AddCmd(rootCmd *cobra.Command) {
	rootCmd.AddCommand(p.Cmd())
}

func (p *PluginBase) AddRPC(c RPCServer) {
	p.RPC(c)
}
