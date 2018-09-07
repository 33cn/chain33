package pluginmgr

import (
	"github.com/spf13/cobra"
	"gitlab.33.cn/chain33/chain33/types"
)

func Init() {
	pluginMgr.init()
}

func DecodeTx(tx *types.Transaction) interface{} {
	return pluginMgr.decodeTx(tx)
}

func RegisterPlugin(p Plugin) bool {
	return pluginMgr.registerPlugin(p)
}

func AddCmd(rootCmd *cobra.Command) {
	pluginMgr.addCmd(rootCmd)
}

func AddRPC(s RPCServer) {
	pluginMgr.addRPC(s)
}
