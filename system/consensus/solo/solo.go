// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package solo solo共识挖矿
package solo

import (
	"encoding/hex"
	"time"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	drivers "github.com/33cn/chain33/system/consensus"
	cty "github.com/33cn/chain33/system/dapp/coins/types"
	"github.com/33cn/chain33/types"
)

var slog = log.New("module", "solo")

// Client 客户端
type Client struct {
	*drivers.BaseClient
	subcfg    *subConfig
	sleepTime time.Duration
}

func init() {
	drivers.Reg("solo", New)
	drivers.QueryData.Register("solo", &Client{})
}

type subConfig struct {
	Genesis          string `json:"genesis"`
	GenesisBlockTime int64  `json:"genesisBlockTime"`
	WaitTxMs         int64  `json:"waitTxMs"`
	BenchMode        bool   `json:"benchMode"`
}

// New new
func New(cfg *types.Consensus, sub []byte) queue.Module {
	c := drivers.NewBaseClient(cfg)
	var subcfg subConfig
	if sub != nil {
		types.MustDecode(sub, &subcfg)
	}
	if subcfg.WaitTxMs == 0 {
		subcfg.WaitTxMs = 1000
	}
	if subcfg.Genesis == "" {
		subcfg.Genesis = cfg.Genesis
	}
	if subcfg.GenesisBlockTime == 0 {
		subcfg.GenesisBlockTime = cfg.GenesisBlockTime
	}
	solo := &Client{
		BaseClient: c,
		subcfg:     &subcfg,
		sleepTime:  time.Duration(subcfg.WaitTxMs) * time.Millisecond,
	}
	c.SetChild(solo)
	return solo
}

// Close close
func (client *Client) Close() {
	slog.Info("consensus solo closed")
}

// GetGenesisBlockTime 获取创世区块时间
func (client *Client) GetGenesisBlockTime() int64 {
	return client.subcfg.GenesisBlockTime
}

// CreateGenesisTx 创建创世交易
func (client *Client) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	cfg := client.GetAPI().GetConfig()
	tx.Execer = []byte(cfg.GetCoinExec())
	tx.To = client.subcfg.Genesis
	//gen payload
	g := &cty.CoinsAction_Genesis{}
	g.Genesis = &types.AssetsGenesis{}
	g.Genesis.Amount = 1e8 * cfg.GetCoinPrecision()
	tx.Payload = types.Encode(&cty.CoinsAction{Value: g, Ty: cty.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

// ProcEvent false
func (client *Client) ProcEvent(msg *queue.Message) bool {
	return false
}

// CheckBlock solo没有交易时返回错误
func (client *Client) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	if len(current.Block.Txs) == 0 {
		return types.ErrEmptyTx
	}
	return nil
}

// CreateBlock 创建区块
func (client *Client) CreateBlock() {
	issleep := true
	types.AssertConfig(client.GetAPI())
	cfg := client.GetAPI().GetConfig()
	beg := types.Now()
	for {

		if client.IsClosed() {
			break
		}
		if !client.IsMining() || !client.IsCaughtUp() {
			time.Sleep(client.sleepTime)
			continue
		}
		if issleep {
			time.Sleep(client.sleepTime)
		}
		lastBlock := client.GetCurrentBlock()
		maxTxNum := int(cfg.GetP(lastBlock.Height + 1).MaxTxNumber)
		txs := client.RequestTx(maxTxNum, nil)
		txs = client.CheckTxDup(txs)

		// 为方便测试，设定基准测试模式，每个块交易数保持恒定，为配置的最大交易数
		if len(txs) == 0 || (client.subcfg.BenchMode && len(txs) < maxTxNum) {
			if len(txs) > 1000 {
				log.Info("======SoloWaitMoreTxs======", "currTxNum", len(txs))
			}
			issleep = true
			continue
		}
		issleep = false
		waitTxCost := types.Since(beg)
		beg = types.Now()
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash(cfg)
		newblock.Height = lastBlock.Height + 1
		client.AddTxsToBlock(&newblock, txs)
		//solo 挖矿固定难度
		newblock.Difficulty = cfg.GetP(0).PowLimitBits
		//需要首先对交易进行排序然后再计算TxHash
		if cfg.IsFork(newblock.GetHeight(), "ForkRootHash") {
			newblock.Txs = types.TransactionSort(newblock.Txs)
		}
		newblock.BlockTime = types.Now().Unix()

		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		log.Info("SoloNewBlock", "height", newblock.Height, "txs", len(txs), "waitTxs", waitTxCost, "writeBlock", types.Since(beg))
		beg = types.Now()
		//判断有没有交易是被删除的，这类交易要从mempool 中删除
		if err != nil {
			log.Error("SoloNewBlock", "height", newblock.Height, "hash", hex.EncodeToString(newblock.Hash(cfg)), "err", err)
			issleep = true
			continue
		}
	}
}

// CmpBestBlock 比较newBlock是不是最优区块
func (client *Client) CmpBestBlock(newBlock *types.Block, cmpBlock *types.Block) bool {
	return false
}
