package ticket

import (
	"time"

	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var slog = log.New("module", "ticket")

type TicketClient struct {
	*drivers.BaseClient
}

func New(cfg *types.Consensus) *TicketClient {
	c := drivers.NewBaseClient(cfg)
	t := &TicketClient{c}
	c.SetChild(t)
	return t
}

func (client *TicketClient) Close() {
	log.Info("consensus ticket closed")
}

func (client *TicketClient) CreateGenesisTx() (ret []*types.Transaction) {
	tx1 := types.Transaction{}
	tx1.Execer = []byte("coins")

	//给hotkey 10000 个币，作为miner的手续费
	tx1.To = client.Cfg.HotkeyAddr
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{Amount: 1e4 * types.Coin}
	tx1.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx1)

	//给ticket 合约打 3亿 个币

	tx2 := types.Transaction{}

	tx2.Execer = []byte("coins")
	tx2.To = execdrivers.ExecAddress("ticket").String()
	//gen payload
	g = &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{3e8 * types.Coin, client.Cfg.Genesis}
	tx2.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx2)

	//产生30w张初始化ticket
	tx3 := types.Transaction{}
	tx3.Execer = []byte("ticket")
	tx3.To = execdrivers.ExecAddress("ticket").String()

	gticket := &types.TicketAction_Genesis{}
	gticket.Genesis = &types.TicketGenesis{client.Cfg.HotkeyAddr, client.Cfg.Genesis, 300000}

	tx3.Payload = types.Encode(&types.TicketAction{Value: gticket, Ty: types.TicketActionGenesis})
	ret = append(ret, &tx3)
	return
}

func (client *TicketClient) CreateBlock() {
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
