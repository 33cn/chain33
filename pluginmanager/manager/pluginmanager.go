package manager

import (
	"fmt"
	"strings"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/pluginmanager/plugin"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	mgrlog    = log15.New("plugin", "plugin.manager")
	pluginMgr = &pluginManager{}
)

func init() {
	pluginMgr.init()
}

type pluginManager struct {
	pluginItems map[string]plugin.Plugin
}

func (mgr *pluginManager) init() {
	mgr.pluginItems = make(map[string]plugin.Plugin, 0)
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

func getRealExecuteName(packageName, exectorName string) string {
	// TODO: 需要实现兼容老系统，并且又能照顾新规则的执行器名称处理功能
	return fmt.Sprintf("%s.%s", packageName, exectorName)
}

func (mgr *pluginManager) decodeTx(tx *types.Transaction) interface{} {
	execName := string(tx.Execer)
	for _, item := range mgr.pluginItems {
		if getRealExecuteName(item.GetPackageName(), item.GetExecutorName()) != execName {
			continue
		}
		return item.DecodeTx(tx)
	}

	return nil
}
