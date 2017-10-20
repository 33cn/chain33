package blockchain

import (
	. "common"
	"common/db"
	"errors"
	"types"
)

var txindexlog = chainlog.New("submodule", "txindex")

// TxIndex is the simplest possible indexer, backed by Key-Value storage (levelDB).
// It could only index transaction by its identifier.
type TxIndex struct {
	store db.DB
}

// NewTxIndex returns new instance of TxIndex.
func NewTxIndex(store db.DB) *TxIndex {
	return &TxIndex{store: store}
}

// Get gets transaction from the TxIndex storage and returns it or nil if the transaction is not found.
func (txi *TxIndex) Get(hash []byte) (*types.TxResult, error) {
	if len(hash) == 0 {
		err := errors.New("input hash is null")
		return nil, err
	}

	rawBytes := txi.store.Get(hash)
	if rawBytes == nil {
		return nil, nil
	}

	var txresult types.TxResult
	err := Decode(rawBytes, &txresult)
	if err != nil {
		return nil, err
	}
	return &txresult, nil
}

func (txi *TxIndex) indexTxs(block *types.Block) {

	storeBatch := txi.store.NewBatch()
	txlen := len(block.Txs)

	for index := 0; index < txlen; index++ {
		// 计算txhash值
		txbyte, err := Encode(block.Txs[index])
		if err != nil {
			txindexlog.Error("indexTxs Encode tx err", "Height", block.Height, "index", index)
			continue
		}
		txhash := BytesToHash(txbyte)
		txhashbyte := txhash.Bytes()

		var txresult types.TxResult
		txresult.Height = block.Height
		txresult.Index = int32(index + 1)
		txresult.Tx = block.Txs[index]

		txresultbyte, err := Encode(txresult)
		if err != nil {
			txindexlog.Error("indexTxs Encode txresult err", "Height", block.Height, "index", index)
			continue
		}
		storeBatch.Set(txhashbyte, txresultbyte)
		//txindexlog.Debug("indexTxs Set txresult", "Height", block.Height, "index", index, "txhashbyte", txhashbyte)
	}
	storeBatch.Write()
}
