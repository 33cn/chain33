package solo

import (
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/merkle"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
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

func (client *SoloClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error) {
	//exec block
	if block.Height == 0 {
		block.Difficulty = types.GetP(0).PowLimitBits
	}
	blockdetail, deltx, err := util.ExecBlock(client.GetQueueClient(), prevHash, block, false, true)
	if err != nil { //never happen
		return nil, deltx, err
	}
	if len(blockdetail.Block.Txs) == 0 {
		return nil, deltx, types.ErrNoTx
	}
	return blockdetail, deltx, nil
}

func (client *SoloClient) CreateBlock() {
	issleep := true
	for {
		if !client.IsMining() || !client.IsCaughtUp() {
			time.Sleep(time.Second)
			continue
		}
		if issleep {
			time.Sleep(time.Second)
		}
		lastBlock := client.GetCurrentBlock()
		txs := client.RequestTx(int(types.GetP(lastBlock.Height+1).MaxTxNumber), nil)
		if len(txs) == 0 {
			issleep = true
			continue
		}
		issleep = false
		//check dup
		//txs = client.CheckTxDup(txs)
		var newblock types.Block
		newblock.ParentHash = lastBlock.Hash()
		newblock.Height = lastBlock.Height + 1
		newblock.Txs = txs
		//solo 挖矿固定难度
		newblock.Difficulty = types.GetP(0).PowLimitBits
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		newblock.BlockTime = time.Now().Unix()
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
