package game

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/game/executor"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	"gitlab.33.cn/chain33/chain33/pluginmanager/manager"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugin"
	"gitlab.33.cn/chain33/chain33/types"
)

var gGamePlugin *gamePlugin

func init() {
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
	return executor.GetName()
}

func (p *gamePlugin) InitExecutor() {
	executor.Init()

	drivers.Register(executor.GetName(), executor.NewGame, 0)
}

func (p *gamePlugin) DecodeTx(tx *types.Transaction) interface{} {
	return executor.DecodeTx(tx)
}
