package bizpolicy

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/wallet/walletbiz"
)

// WalletPolicy 细分钱包业务逻辑的街口
type WalletBizPolicy interface {
	Init(walletBiz walletbiz.WalletBiz)
	OnRecvQueueMsg(msg *queue.Message) error
}
