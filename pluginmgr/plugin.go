package pluginmgr

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/rpc/types"
)

//
type Plugin interface {
	// 获取整个插件的包名，用以计算唯一值、做前缀等
	GetName() string
	// 获取插件中执行器名
	GetExecutorName() string
	// 初始化执行器时会调用该接口
	InitExec(sub map[string][]byte)
	AddCmd(rootCmd *cobra.Command)
	AddRPC(s types.RPCServer)
}
