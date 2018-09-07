package pluginmgr

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/types"
)

type PluginBase struct {
}

func (p *PluginBase) GetName() string {
	return ""
}

func (p *PluginBase) GetExecutorName() string {
	return ""
}

func (p *PluginBase) Init() {
}

func (p *PluginBase) DecodeTx(tx *types.Transaction) interface{} {
	return nil
}

func (p *PluginBase) AddCmd(rootCmd *cobra.Command) {
}

func (p *PluginBase) AddRPC(c RPCServer) {
}
