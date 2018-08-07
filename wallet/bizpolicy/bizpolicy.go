package bizpolicy

import (
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/wallet/walletoperate"
)

// WalletPolicy 细分钱包业务逻辑的街口
type WalletBizPolicy interface {
	// Init 初始化钱包业务策略，在使用前调用
	Init(walletBiz walletoperate.WalletOperate)
	// OnRecvQueueMsg 钱包收到消息队列的消息后，分发给钱包业务逻辑进行处理
	OnRecvQueueMsg(msg *queue.Message) (bool, error)
	// OnAddBlockTx 当区块被增加确认时调用本函数
	// block： 被增加的区块详细信息
	// tx: 区块增加的交易信息
	// index: 交易信息在区块上的索引为止，从0开始计数
	// dbbatch: 数据库批量操作接口
	OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch)
	// OnDeleteBlockTx 当区块被回退删除时调用本函数
	// block： 被回退的区块详细信息
	// tx: 区块回退的交易信息
	// index: 交易信息在区块上的索引为止，从0开始计数
	// dbbatch: 数据库批量操作接口
	OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch)
	OnAddBlockFinish()
	OnDeleteBlockFinish()
}
