//对SC进行注册
package siacoin

import (
	"gitlab.33.cn/wallet/bipwallet/transformer"
)

func init() {
	//注册
	transformer.Register("SC", &ScTransformer{})
}
