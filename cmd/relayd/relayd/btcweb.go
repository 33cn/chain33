package relayd

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	log "github.com/inconshreveable/log15"
	"github.com/valyala/fasthttp"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
)

type btcWeb struct {
	urlRoot    string
	httpClient *fasthttp.Client
}

func NewBtcWeb() (BtcClient, error) {
	b := &btcWeb{
		urlRoot:    "https://blockchain.info",
		httpClient: &fasthttp.Client{TLSConfig: &tls.Config{InsecureSkipVerify: true}},
	}
	return b, nil
}

func (b *btcWeb) Start() error {
	return nil
}

func (b *btcWeb) Stop() error {
	return nil
}

func (b *btcWeb) GetBlockHeader(height uint64) (*ty.BtcHeader, error) {
	block, err := b.getBlock(height)
	if err != nil {
		return nil, err
	}
	return block.BtcHeader(), nil
}

func (b *btcWeb) getBlock(height uint64) (*Block, error) {
	if height < 0 {
		return nil, errors.New("height < 0")
	}
	url := fmt.Sprintf("%s/block-height/%d?format=json", b.urlRoot, height)
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

func (b *btcWeb) GetLatestBlock() (*chainhash.Hash, uint64, error) {
	url := b.urlRoot + "/latestblock"
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

func (b *btcWeb) Ping() {
	hash, height, err := b.GetLatestBlock()
	if err != nil {
		log.Error("btcWeb ping", "error", err)
	}
	log.Info("btcWeb ping", "latest Hash: ", hash.String(), "latest height", height)
}

func (b *btcWeb) GetTransaction(hash string) (*ty.BtcTransaction, error) {
	url := b.urlRoot + "/rawtx/" + hash
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

func (b *btcWeb) GetSPV(height uint64, txHash string) (*ty.BtcSpv, error) {
	block, err := b.getBlock(height)
	if err != nil {
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
	spv := &ty.BtcSpv{
		Hash:        txHash,
		Time:        block.Time,
		Height:      block.Height,
		BlockHash:   block.Hash,
		TxIndex:     txIndex,
		BranchProof: proof,
	}
	return spv, nil
}

func (b *btcWeb) requestUrl(url string) ([]byte, error) {
	status, body, err := b.httpClient.Get(nil, url)
	if err != nil {
		return nil, err
	}
	if status != fasthttp.StatusOK {
		return nil, fmt.Errorf("%d", status)
	}
	return body, nil
}
