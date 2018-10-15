package executor

import (
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/state"
	"gitlab.33.cn/chain33/chain33/types"
)

// 检查合约调用账户是否有充足的金额进行转账交易操作
func CanTransfer(db state.StateDB, sender, recipient common.Address, amount uint64) bool {
	return db.CanTransfer(sender.String(), recipient.String(), amount)
}

// 在内存数据库中执行转账操作（只修改内存中的金额）
// 从外部账户地址到合约账户地址
func Transfer(db state.StateDB, sender, recipient common.Address, amount uint64) bool {
	return db.Transfer(sender.String(), recipient.String(), amount)
}

// 获取制定高度区块的哈希
func GetHashFn(api client.QueueProtocolAPI) func(blockHeight uint64) common.Hash {
	return func(blockHeight uint64) common.Hash {
		if api != nil {
			reply, err := api.GetBlockHash(&types.ReqInt{int64(blockHeight)})
			if nil != err {
				log.Error("Call GetBlockHash Failed.", err)
			}
			return common.BytesToHash(reply.Hash)
		}
		return common.Hash{}
	}
}
