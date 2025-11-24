package types

import (
	"math/big"

	etypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// BlockDetailToEthBlock chain33 blockdetails transfer to  eth block format
func BlockDetailToEthBlock(details *types.BlockDetails, cfg *types.Chain33Config, full bool) (*Block, error) {
	var block Block
	var header Header
	fullblock := details.GetItems()[0]
	header.Time = hexutil.Uint64(fullblock.GetBlock().GetBlockTime())
	header.Number = (*hexutil.Big)(big.NewInt(fullblock.GetBlock().Height))
	header.TxHash = common.BytesToHash(fullblock.GetBlock().GetHeader(cfg).TxHash)
	header.Difficulty = (*hexutil.Big)(big.NewInt(int64(fullblock.GetBlock().GetDifficulty())))
	header.ParentHash = common.BytesToHash(fullblock.GetBlock().ParentHash)
	header.Root = common.BytesToHash(fullblock.GetBlock().GetStateHash())
	header.Coinbase = common.HexToAddress(fullblock.GetBlock().GetTxs()[0].From())
	//header.GasUsed=
	//暂不支持ReceiptHash,UncleHash
	//header.ReceiptHash=
	header.UncleHash = etypes.EmptyUncleHash

	//处理交易
	txs, fee, err := TxsToEthTxs(common.BytesToHash(fullblock.GetBlock().Hash(cfg)), fullblock.GetBlock().Height, fullblock.GetBlock().GetTxs(), cfg, full)
	if err != nil {
		return nil, err
	}
	header.GasUsed = hexutil.Uint64(fee)
	header.GasLimit = header.GasUsed
	block.Hash = common.BytesToHash(fullblock.GetBlock().Hash(cfg)).Hex()

	block.Header = &header
	block.Uncles = []*Header{}
	block.Transactions = txs
	return &block, nil
}

// BlockHeaderToEthHeader transfer chain33 header to eth header
func BlockHeaderToEthHeader(cHeader *types.Header) (*Header, error) {
	var header Header
	header.Time = hexutil.Uint64(cHeader.GetBlockTime())
	header.Number = (*hexutil.Big)(big.NewInt(cHeader.Height))
	header.TxHash = common.BytesToHash(cHeader.TxHash)
	header.Difficulty = (*hexutil.Big)(big.NewInt(int64(cHeader.GetDifficulty())))
	header.ParentHash = common.BytesToHash(cHeader.ParentHash)
	header.Root = common.BytesToHash(cHeader.GetStateHash())
	return &header, nil
}
