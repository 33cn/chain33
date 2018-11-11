package manage

import (
	"github.com/33cn/chain33/pluginmgr"
	"github.com/33cn/chain33/system/dapp/manage/commands"
	"github.com/33cn/chain33/system/dapp/manage/executor"
	"github.com/33cn/chain33/system/dapp/manage/types"
)

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     types.ManageX,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.ConfigCmd,
		RPC:      nil,
	})
}
