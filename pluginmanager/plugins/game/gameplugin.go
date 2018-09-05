package game

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/pluginmanager/manager"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugin"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugins/game/types"

	"gitlab.33.cn/chain33/chain33/pluginmanager/plugins/game/executor"
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
	return types.PackageName
}

func (p *gamePlugin) GetExecutorName() string {
	return executor.GetName()
}

func (p *gamePlugin) InitExecutor() {
	executor.Init()

	drivers.Register(executor.GetName(), executor.NewGame, 0)
}