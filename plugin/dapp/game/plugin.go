package game

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/game/executor"
	gt "gitlab.33.cn/chain33/chain33/plugin/dapp/game/types"
	"gitlab.33.cn/chain33/chain33/pluginmgr"
	"gitlab.33.cn/chain33/chain33/types"
)

var gGamePlugin *gamePlugin

func init() {
	gGamePlugin = &gamePlugin{}
	pluginmgr.RegisterPlugin(gGamePlugin)
}

type gamePlugin struct {
	pluginmgr.PluginBase
}

func (p *gamePlugin) GetName() string {
	return gt.PackageName
}

func (p *gamePlugin) GetExecutorName() string {
	return executor.GetName()
}

func (p *gamePlugin) Init() {
	executor.Init()
	drivers.Register(executor.GetName(), executor.NewGame, 0)
}

func (p *gamePlugin) DecodeTx(tx *types.Transaction) interface{} {
	return executor.DecodeTx(tx)
}
