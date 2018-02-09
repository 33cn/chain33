package solo

import (
	"time"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/util"
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
	g.Genesis = &types.CoinsGenesis{}
	g.Genesis.Amount = 1e8 * types.Coin
	tx.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx)
	return
}

func (client *SoloClient) ProcEvent(msg queue.Message) {

}

//solo 不检查任何的交易
func (client *SoloClient) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	return nil
}

func (client *SoloClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, error) {
	//exec block
	if block.Height == 0 {
		block.Difficulty = types.PowLimitBits
	}
	blockdetail, err := util.ExecBlock(client.GetQueue(), prevHash, block, false)
	if err != nil { //never happen
		return nil, err
	}
	if len(blockdetail.Block.Txs) == 0 {
		return nil, types.ErrNoTx
	}
	return blockdetail, nil
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
		newblock.Difficulty = types.PowLimitBits
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
