package plugin

import "gitlab.33.cn/chain33/chain33/types"

//
type Plugin interface {
	// 获取整个插件的包名
	GetPackageName() string
	// 获取插件中执行器名
	GetExecutorName() string
	InitExecutor()
	DecodeTx(tx *types.Transaction) interface{}
}
