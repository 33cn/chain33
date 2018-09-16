package pluginmgr

import (
	"sync"

	"github.com/spf13/cobra"
)

var pluginItems = make(map[string]Plugin)

var once = &sync.Once{}

func InitExec() {
	once.Do(func() {
		for _, item := range pluginItems {
			item.InitExec()
		}
	})
}

func Register(p Plugin) {
	if p == nil {
		panic("plugin param is nil" + p.GetName())
	}
	packageName := p.GetName()
	if len(packageName) == 0 {
		panic("plugin package name is empty")
	}
	if _, ok := pluginItems[packageName]; ok {
		panic("execute plugin item is existed. name = " + packageName)
	}
	pluginItems[packageName] = p
}

func AddCmd(rootCmd *cobra.Command) {
	for _, item := range pluginItems {
		item.AddCmd(rootCmd)
	}
}

func AddRPC(s RPCServer) {
	for _, item := range pluginItems {
		item.AddRPC(s)
	}
}
