package relayd

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/valyala/fasthttp"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/types"
)

type BtcWeb struct {
	httpClient *fasthttp.Client
}

func NewBtcWeb() BtcClient {
	b := &BtcWeb{
		httpClient: &fasthttp.Client{TLSConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	return b
}

func (b *BtcWeb) Start() error {
	return nil
}

func (b *BtcWeb) Stop() error {
	return nil
}

func (b *BtcWeb) GetBlockHeader(height uint64) (*types.BtcHeader, error) {
	block, err := b.getBlock(height)
	if err != err {
		return nil, err
	}
	return block.Header.BtcHeader(), nil
}

func (b *BtcWeb) getBlock(height uint64) (*Block, error) {
	if height < 0 {
		return nil, errors.New("height < 0")
	}
	url := fmt.Sprintf("https://blockchain.info/block-height/%d?format=json", height)
	data, err := b.requestUrl(url)
	if err != nil {
		return nil, err
	}
	var blocks = Blocks{}
	err = json.Unmarshal(data, &blocks)
	if err != nil {
		return nil, err
	}
	block := blocks.Blocks[0]
	return &block, nil
}

func (b *BtcWeb) GetLatestBlock() (*chainhash.Hash, uint64, error) {
	url := "https://blockchain.info/latestblock"
	data, err := b.requestUrl(url)
	if err != nil {
		return nil, 0, err
	}
	var blocks = LatestBlock{}
	err = json.Unmarshal(data, &blocks)
	if err != nil {
		return nil, 0, err
	}

	hash, err := chainhash.NewHashFromStr(blocks.Hash)
	if err != nil {
		return nil, 0, err
	}

	return hash, blocks.Height, nil
}

func (b *BtcWeb) GetTransaction(hash string) (*types.BtcTransaction, error) {
	url := "https://blockchain.info/rawtx/" + hash
	data, err := b.requestUrl(url)
	if err != nil {
		return nil, err
	}
	var tx = TransactionResult{}
	err = json.Unmarshal(data, &tx)
	if err != nil {
		return nil, err
	}
	return tx.BtcTransaction(), nil
}

func (b *BtcWeb) GetSPV(height uint64, txHash string) (*types.BtcSpv, error) {
	block, err := b.getBlock(height)
	if err != err {
		return nil, err
	}
	var txIndex uint32
	txs := make([][]byte, 0, len(block.Tx))
	for index, tx := range block.Tx {
		if txHash == tx.Hash {
			txIndex = uint32(index)
		}
		hash, err := merkle.NewHashFromStr(tx.Hash)
		if err != nil {
			return nil, err
		}
		txs = append(txs, hash.CloneBytes())
	}
	proof := merkle.GetMerkleBranch(txs, txIndex)
	spv := &types.BtcSpv{
		Hash:        txHash,
		Time:        block.Time,
		Height:      block.Height,
		BlockHash:   block.Hash,
		TxIndex:     txIndex,
		BranchProof: proof,
	}
	return spv, nil
}

func (b *BtcWeb) requestUrl(url string) ([]byte, error) {
	status, body, err := b.httpClient.Get(nil, url)
	if err != nil {
		return nil, err
	}
	if status != fasthttp.StatusOK {
		return nil, fmt.Errorf("%d", status)
	}
	return body, nil
}
