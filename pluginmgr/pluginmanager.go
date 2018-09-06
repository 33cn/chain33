package pluginmgr

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var (
	mgrlog    = log15.New("plugin", "plugin.manager")
	pluginMgr *pluginManager
)

func init() {
	pluginMgr = newMgr()
}

type pluginManager struct {
	pluginItems map[string]Plugin
}

//New new pluginManager
func newMgr() *pluginManager {
	mgr := &pluginManager{}
	mgr.pluginItems = make(map[string]Plugin)
	return mgr
}

func (mgr *pluginManager) registerPlugin(p Plugin) bool {
	if p == nil {
		mgrlog.Error("plugin param is nil")
		return false
	}
	packageName := p.GetName()
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

func (mgr *pluginManager) init() {
	for _, item := range mgr.pluginItems {
		item.Init()
	}
}

func getRealExecuteName(packageName, exectorName string) string {
	// TODO: 需要实现兼容老系统，并且又能照顾新规则的执行器名称处理功能
	return fmt.Sprintf("%s.%s", packageName, exectorName)
}

func (mgr *pluginManager) decodeTx(tx *types.Transaction) interface{} {
	execName := string(tx.Execer)
	for _, item := range mgr.pluginItems {
		if getRealExecuteName(item.GetName(), item.GetExecutorName()) != execName {
			continue
		}
		return item.DecodeTx(tx)
	}

	return nil
}

func (mgr *pluginManager) addCmd(rootCmd *cobra.Command) {
	for _, item := range mgr.pluginItems {
		item.AddCmd(rootCmd)
	}
}
