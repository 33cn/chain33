package manager

import (
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugin"
	"gitlab.33.cn/chain33/chain33/types"
	"strings"
)

var (
	mgrlog    = log15.New("plugin", "plugin.manager")
	pluginMgr = &pluginManager{}
)

func init()  {
	pluginMgr.init()
}

type executorPluginItem struct {
	name    string
	creator drivers.DriverCreate
	height  int64
}

type pluginManager struct {
	pluginItems map[string]plugin.Plugin

	execPluginItems map[string]*executorPluginItem
}

func (mgr *pluginManager) init() {
	mgr.pluginItems = make(map[string]plugin.Plugin, 0)

	mgr.execPluginItems = make(map[string]*executorPluginItem, 0)
}

func (mgr *pluginManager) registerPlugin(p plugin.Plugin) bool {
	if p == nil {
		mgrlog.Error("plugin param is nil")
		return false
	}
	packageName := p.GetPackageName()
	if len(packageName) == 0 {
		mgrlog.Error("plugin package name is empty")
		return false
	}
	if _, ok := mgr.pluginItems[packageName]; ok {
		mgrlog.Error("execute plugin item is existed.", "package name", packageName)
		return false
	}
	mgr.pluginItems[packageName] = p
	return true
}

func (mgr *pluginManager) initExecutor() {
	for _, item := range mgr.pluginItems {
		item.InitExecutor()
	}
}

func realExec(txExec string) string {
	if strings.HasPrefix(txExec, "user.p.") {
		execSplit := strings.Split(txExec, ".")
		return execSplit[len(execSplit)-1]
	}
	return txExec
}

func (mgr *pluginManager) decodeTx(tx *types.Transaction) interface{} {
	key := realExec(string(tx.Execer))
	if _, ok := mgr.execPluginItems[key]; ok {
		// TODO: 通过插件接口，调用插件内部的功能
	}
	
	return nil
}
