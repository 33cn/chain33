package relayd

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/valyala/fasthttp"
	"gitlab.33.cn/chain33/chain33/common/merkle"
)

type BTCWeb struct {
	httpClient *fasthttp.Client
}

func NewBTCWeb() *BTCWeb {
	b := &BTCWeb{
		httpClient: &fasthttp.Client{TLSConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	return b
}

func (b *BTCWeb) GetBlockHeader(height uint64) (*Header, error) {
	block, err := b.GetBlock(height)
	if err != err {
		return nil, err
	}
	return &block.Header, nil
}

func (b *BTCWeb) GetBlock(height uint64) (*Block, error) {
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

func (b *BTCWeb) GetLatestBlock() (*LatestBlock, error) {
	url := "https://blockchain.info/latestblock"
	data, err := b.requestUrl(url)
	if err != nil {
		return nil, err
	}
	var blocks = LatestBlock{}
	err = json.Unmarshal(data, &blocks)
	if err != nil {
		return nil, err
	}
	return &blocks, nil
}

func (b *BTCWeb) GetTransaction(hash string) (*TransactionResult, error) {
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
	return &tx, nil
}

func (b *BTCWeb) GetSPV(height uint64, txHash string) (*SPVInfo, error) {
	block, err := b.GetBlock(height)
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
	spv := &SPVInfo{
		Hash:        txHash,
		Time:        block.Time,
		Height:      block.Height,
		BlockHash:   block.Hash,
		TxIndex:     uint64(txIndex),
		BranchProof: proof,
	}
	return spv, nil
}

func (b *BTCWeb) requestUrl(url string) ([]byte, error) {
	status, body, err := b.httpClient.Get(nil, url)
	if err != nil {
		return nil, err
	}
	if status != fasthttp.StatusOK {
		return nil, errors.New(fmt.Sprintf("%d", status))
	}
	return body, nil
}
