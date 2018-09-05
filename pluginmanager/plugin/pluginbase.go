package plugin

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/types"
)

type PluginBase struct {
}

func (p *PluginBase) GetPackageName() string {
	return ""
}

func (p *PluginBase) GetExecutorName() string {
	return ""
}

func (p *PluginBase) InitExecutor() {
}

func (p *PluginBase) DecodeTx(tx *types.Transaction) interface{} {
	return nil
}

func (p *PluginBase) AddCustomCommand(rootCmd *cobra.Command) {

}
