package types

import (
	"context"
	"errors"

	ctypes "github.com/33cn/chain33/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

// Filter 过滤器
type Filter struct {
	grpcCli     ctypes.Chain33Client
	cfg         *ctypes.Chain33Config
	bloom       types.Bloom
	option      *FilterQuery
	filterTopic bool
}

// NewFilter 创建filter 对象
func NewFilter(gcli ctypes.Chain33Client, cfg *ctypes.Chain33Config, option *FilterQuery) (*Filter, error) {
	var bloom types.Bloom
	filterTopic := false
	if option != nil {
		if len(option.Topics) != 0 {
			filterTopic = true
			for _, t := range option.Topics {

				switch topic := t.(type) {
				case string:
					bloom.Add(common.HexToHash(topic).Bytes())
				case []interface{}:
					for _, rtopic := range topic {
						if t, ok := rtopic.(string); ok {
							bloom.Add(common.HexToHash(t).Bytes())
						}
					}

				default:
					return nil, errors.New("no support topic type")

				}
			}
		}

		switch option.Address.(type) {
		case []interface{}:
			for _, addr := range option.Address.([]interface{}) {
				if eaddr, ok := addr.(string); ok {
					bloom.Add(common.HexToAddress(eaddr).Bytes())
				}

			}
		case string:
			bloom.Add(common.HexToAddress(option.Address.(string)).Bytes())
		}
	}

	return &Filter{
		grpcCli:     gcli,
		cfg:         cfg,
		bloom:       bloom,
		option:      option,
		filterTopic: filterTopic,
	}, nil
}

// FilterBlockDetail 根据blockhash 过滤evmlog
func (f *Filter) FilterBlockDetail(blockHashes [][]byte) ([]*EvmLog, error) {
	blockdetails, err := f.grpcCli.GetBlockByHashes(context.Background(), &ctypes.ReqHashes{Hashes: blockHashes})
	if err != nil {
		return nil, err
	}
	var evmlogs []*EvmLog
	var receipts []*Receipt
	for _, detail := range blockdetails.GetItems() {
		for _, tx := range detail.Block.GetTxs() {
			txdetail, err := f.grpcCli.GetTransactionByHashes(context.Background(), &ctypes.ReqHashes{Hashes: [][]byte{tx.Hash()}})
			if err != nil {
				continue
			}
			_, receipt, err := TxDetailsToEthReceipts(txdetail, common.BytesToHash(detail.Block.Hash(f.cfg)), f.cfg)
			if err != nil {
				continue
			}
			receipts = append(receipts, receipt...)
		}
	}

	logs := f.filterReceipt(receipts)
	evmlogs = append(evmlogs, logs...)
	return evmlogs, nil
}

func (f *Filter) filterReceipt(receipts []*Receipt) (evmlogs []*EvmLog) {
	for _, receipt := range receipts {
		if receipt.ContractAddress != nil && f.bloomFilterAddress(receipt.ContractAddress.Bytes()) {
			if len(receipt.Logs) != 0 {
				for index, log := range receipt.Logs {
					var info EvmLog
					if len(log.Topics) != 0 {
						if f.bloomFilterTopics(log.Topics[0].Bytes()) { // 目前只过滤函数方法签名
							info.Topics = log.Topics
							info.Data = log.Data
							info.Address = *receipt.ContractAddress
							info.TxIndex = receipt.TransactionIndex
							info.Index = hexutil.Uint(index)
							info.BlockHash = receipt.BlockHash
							info.TxHash = receipt.TxHash
							info.BlockNumber = hexutil.Uint64(receipt.BlockNumber.ToInt().Int64())
							evmlogs = append(evmlogs, &info)
						}
					}

				}
			}

		}

	}

	return evmlogs
}

// FilterEvmTxLogs 过滤EVMTxLogPerBlk
func (f *Filter) FilterEvmTxLogs(logs *ctypes.EVMTxLogPerBlk) (evmlogs []*EvmLog) {

	for _, txlog := range logs.TxAndLogs {
		if f.bloomFilterAddress(common.HexToAddress(txlog.GetTx().GetTo()).Bytes()) {
			for j, tlog := range txlog.GetLogsPerTx().GetLogs() {
				var topics [][]byte
				var info EvmLog
				if f.bloomFilterTopics(tlog.Topic[0]) {
					topics = append(topics, tlog.Topic...)
				} else {
					continue
				}

				info.Index = hexutil.Uint(j)
				for _, topic := range topics {
					info.Topics = append(info.Topics, common.BytesToHash(topic))
				}

				to := common.HexToAddress(txlog.GetTx().GetTo())
				info.Address = to
				info.Index = hexutil.Uint(j)
				info.BlockHash = common.BytesToHash(logs.GetBlockHash()) //hexutil.Encode(logs.BlockHash)
				info.TxHash = common.BytesToHash(txlog.GetTx().Hash())
				info.BlockNumber = hexutil.Uint64(uint64(logs.Height))
				info.Data = tlog.Data
				evmlogs = append(evmlogs, &info)
			}
		}
	}

	return
}

func (f *Filter) bloomFilterAddress(topic []byte) bool {

	return f.bloom.Test(topic) || f.option == nil || f.option != nil && f.option.Address == nil
}

func (f *Filter) bloomFilterTopics(topic []byte) bool {
	return f.bloom.Test(topic) || f.option == nil || f.option != nil && !f.filterTopic

}
