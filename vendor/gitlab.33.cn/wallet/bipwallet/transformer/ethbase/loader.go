//对ETH和ETC进行注册
package ethbase

import (
	"gitlab.33.cn/wallet/bipwallet/transformer"
)

func init() {
	//注册
	transformer.Register("ETH", &EthBaseTransformer{})
	transformer.Register("ETC", &EthBaseTransformer{})
}
