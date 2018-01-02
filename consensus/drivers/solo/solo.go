package solo

import (
	"time"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var slog = log.New("module", "solo")

type SoloClient struct {
	*drivers.BaseClient
}

func New(cfg *types.Consensus) *SoloClient {
	c := drivers.NewBaseClient(cfg)
	solo := &SoloClient{c}
	c.SetChild(solo)
	return solo
}

func (client *SoloClient) Close() {
	log.Info("consensus solo closed")
}

func (client *SoloClient) CreateGenesisTx() (ret []*types.Transaction) {
	var tx types.Transaction
	tx.Execer = []byte("coins")
	tx.To = client.Cfg.Genesis
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{1e8 * types.Coin}
	tx.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

func (client *SoloClient) CreateBlock() {
	issleep := true
	for {
		if !client.IsMining() {
			time.Sleep(time.Second)
			continue
		}
		if issleep {
			time.Sleep(time.Second)
		}
		txs := client.RequestTx()
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		//check dup
		txs = client.CheckTxDup(txs)
		lastBlock := client.GetCurrentBlock()
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		newblock.Txs = txs
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = time.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		if err != nil {
			issleep = true
			continue
		}
	}
}
