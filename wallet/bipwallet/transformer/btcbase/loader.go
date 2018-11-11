//对各个币种进行注册
package btcbase

import (
	"github.com/33cn/chain33/wallet/bipwallet/transformer"
)

//不同币种的前缀版本号
var coin_prefix = map[string][]byte{
	"BTC":  {0x00},
	"BCH":  {0x00},
	"BTY":  {0x00},
	"LTC":  {0x30},
	"ZEC":  {0x1c, 0xb8},
	"USDT": {0x00},
}

func init() {
	//注册
	for name, prefix := range coin_prefix {
		transformer.Register(name, &BtcBaseTransformer{prefix})
	}
}
