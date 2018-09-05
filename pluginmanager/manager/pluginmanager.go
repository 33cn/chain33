package manager

import (
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/executor/drivers"
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
	execPluginItems map[string]*executorPluginItem
}

func (mgr *pluginManager) init() {
	mgr.execPluginItems = make(map[string]*executorPluginItem, 0)
}

func (mgr *pluginManager) registerExecutor(name string, creator drivers.DriverCreate, height int64) bool {
	mgrlog.Info("Register Executor ", "name", name, "forkheight", height)
	if len(name) == 0 || creator == nil || height < 0 {
		mgrlog.Error("invalidate params")
		return false
	}
	if _, ok := mgr.execPluginItems[name]; ok {
		mgrlog.Error("execute plugin item is existed.", "name", name)
		return false
	}
	mgr.execPluginItems[name] = &executorPluginItem{
		name:    name,
		creator: creator,
		height:  height,
	}
	// TODO: 需要初始化类型相关的事件，类似在exectype.Init()中的实现
	return true
}

func (mgr *pluginManager) initExecutor() {
	for _, item := range mgr.execPluginItems {
		drivers.Register(item.name, item.creator, item.height)
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
