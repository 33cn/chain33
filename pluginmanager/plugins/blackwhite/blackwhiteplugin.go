package blackwhite

import (
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/pluginmanager/manager"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugin"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugins/blackwhite/executor"
	"gitlab.33.cn/chain33/chain33/types"
)

var gblackwhitePlugin *blackwhitePlugin

func init() {
	gblackwhitePlugin = &blackwhitePlugin{}
	manager.RegisterPlugin(gblackwhitePlugin)
}

type blackwhitePlugin struct {
	plugin.PluginBase
}

func (p *blackwhitePlugin) GetPackageName() string {
	return "gitlab.33.cn"
}

func (p *blackwhitePlugin) GetExecutorName() string {
	return executor.GetName()
}

func (p *blackwhitePlugin) InitExecutor() {
	// TODO: 这里应该将初始化的内容统一放在一个初始化的地方
	executor.InitTypes()
	executor.SetReciptPrefix()

	drivers.Register(executor.GetName(), executor.NewBlackwhite, types.ForkV25BlackWhite)
}
