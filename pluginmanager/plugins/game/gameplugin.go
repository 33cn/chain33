package game

import (
	"gitlab.33.cn/chain33/chain33/pluginmanager/manager"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugin"
	"gitlab.33.cn/chain33/chain33/types"
	gt "gitlab.33.cn/chain33/chain33/pluginmanager/plugins/game/types"
)

var gGamePlugin *gamePlugin

func init()  {
	gGamePlugin = &gamePlugin{}

	manager.RegisterPlugin(gGamePlugin)
}

type gamePlugin struct {
	plugin.PluginBase
}

func (p *gamePlugin) GetPackageName() string {
	return gt.PackageName
}

func (p *gamePlugin) GetExecutorName() string {
	return types.ExecName(gt.GameX)
}

func (p *gamePlugin) Init() {

}