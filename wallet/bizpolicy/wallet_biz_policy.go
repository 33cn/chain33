package bizpolicy

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/wallet/walletoperate"
)

// WalletPolicy 细分钱包业务逻辑的街口
type WalletBizPolicy interface {
	Init(walletBiz walletoperate.WalletOperate)
	OnRecvQueueMsg(msg *queue.Message) (bool, error)
}
