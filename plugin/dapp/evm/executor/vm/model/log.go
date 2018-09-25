package model

import (
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/plugin/dapp/evm/executor/vm/common"
)

// 合约在日志，对应EVM中的Log指令，可以生成指定的日志信息
// 目前这些日志只是在合约执行完成时进行打印，没有其它用途
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

// 合约日志打印格式
func (log *ContractLog) PrintLog() {
	log15.Debug("!Contract Log!", "Contract address", log.Address.String(), "TxHash", log.TxHash.Hex(), "Log Index", log.Index, "Log Topics", log.Topics, "Log Data", common.Bytes2Hex(log.Data))
}
