package types

import (
	"encoding/hex"
	"fmt"
	"gitlab.33.cn/chain33/chain33/executor/drivers/evm/vm/common"
)

type ContractLog struct {
	// 合约地址
	Address common.Address

	// 对应交易哈希
	TxHash common.Hash

	// 日志序号
	Index int

	// 此合约提供的主题信息
	Topics []common.Hash

	// 日志数据
	Data []byte
}

func (log *ContractLog) String() string {
	item := fmt.Sprintf("!Contract Log! Contract address: %s, TxHash: %x, Log Index: %s, Log Topics: %s, Log Data: %s", log.Address.Str(), log.TxHash.Hex(), log.Index, log.Topics, hex.EncodeToString(log.Data))
	return item
}
