package ed25519base

import (
	"github.com/33cn/chain33/wallet/bipwallet/transformer"
)

//不同币种的前缀版本号
var coinPrefix = map[string][]byte{
	"YCC": {0x00},
}

func init() {
	//注册
	for name, prefix := range coinPrefix {
		transformer.Register(name, &baseTransformer{prefix})
	}
}
