package solo

import (
	"time"

	log "gitlab.33.cn/chain33/chain33/common/log/log15"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/consensus"

	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
)

var slog = log.New("module", "solo")

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
}

func New(cfg *types.Consensus, sub []byte) queue.Module {
	c := drivers.NewBaseClient(cfg)
	var subcfg subConfig
	if sub != nil {
		types.MustDecode(sub, &subcfg)
	}
	if subcfg.WaitTxMs == 0 {
		subcfg.WaitTxMs = 1000
	}
	solo := &Client{c, &subcfg, time.Duration(subcfg.WaitTxMs) * time.Millisecond}
	c.SetChild(solo)
	return solo
}

func (client *Client) Close() {
	slog.Info("consensus solo closed")
}

func (client *Client) GetGenesisBlockTime() int64 {
	return client.subcfg.GenesisBlockTime
}

func (client *Client) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = client.subcfg.Genesis
	//gen payload
	g := &cty.CoinsAction_Genesis{}
	g.Genesis = &types.AssetsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&cty.CoinsAction{Value: g, Ty: cty.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

func (client *Client) ProcEvent(msg queue.Message) bool {
	return false
}

//solo 不检查任何的交易
func (client *Client) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	return nil
}

func (client *Client) CreateBlock() {
	issleep := true
	for {
		if !client.IsMining() || !client.IsCaughtUp() {
			time.Sleep(client.sleepTime)
			continue
		}
		if issleep {
			time.Sleep(client.sleepTime)
		}
		lastBlock := client.GetCurrentBlock()
		txs := client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber), nil)
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		//check dup
		txs = client.CheckTxDup(txs)
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		client.AddTxsToBlock(&newblock, txs)
		//solo 挖矿固定难度
		newblock.Difficulty = types.GetP(0).PowLimitBits
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = types.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		//判断有没有交易是被删除的，这类交易要从mempool 中删除
		if err != nil {
			issleep = true
			continue
		}
	}
}
