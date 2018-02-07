package ticket

import (
	"fmt"
	"math/big"
	"time"

	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var slog = log.New("module", "ticket")

type TicketClient struct {
	*drivers.BaseClient
	//ticket list for miner
	tlist *types.TicketList
}

func New(cfg *types.Consensus) *TicketClient {
	c := drivers.NewBaseClient(cfg)
	t := &TicketClient{c, &types.TicketList{}}
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

func (client *TicketClient) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	//检查第一个笔交易的execs, 以及执行状态
	if len(current.Block.Txs) == 0 {
		return types.ErrEmptyTx
	}
	baseTx := current.Block.Txs[0]
	if string(baseTx.Execer) != "ticket" {
		return types.ErrCoinBaseExecer
	}
	//判断交易类型和执行情况
	var ticketAction types.TicketAction
	err := types.Decode(baseTx.GetPayload(), &ticketAction)
	if err != nil {
		return err
	}
	if ticketAction.GetTy() != types.TicketActionMiner {
		return types.ErrCoinBaseTxType
	}
	//判断交易执行是否OK
	if current.Receipts[0].Ty != types.ExecOk {
		return types.ErrCoinBaseExecErr
	}
	//check reward 的值是否正确
	miner := ticketAction.GetMiner()
	if miner.Reward != types.CoinReward {
		return types.ErrCoinbaseReward
	}
	//通过判断区块的难度difficult
	diff := client.getNextTarget(parent, miner.Bits)
	currentdiff := client.getCurrentTarget(current.Block.BlockTime, miner.TicketId, miner.Modify)
	if currentdiff.Cmp(CompactToBig(miner.Bits)) != 0 {
		return types.ErrCoinBaseTarget
	}
	if currentdiff.Cmp(diff) > 0 {
		return types.ErrCoinBaseTarget
	}
	return nil
}

func (client *TicketClient) getNextTarget(block *types.Block, bits uint32) *big.Int {
	targetBits, err := client.GetNextRequiredDifficulty(block, bits)
	if err != nil {
		panic(err)
	}
	return CompactToBig(targetBits)
}

func (client *TicketClient) getCurrentTarget(blocktime int64, id string, modify []byte) *big.Int {
	s := fmt.Sprint("%d:%s:%x", blocktime, id, modify)
	hash := common.Sha2Sum([]byte(s))
	return HashToBig(hash[:])
}

// calcNextRequiredDifficulty calculates the required difficulty for the block
// after the passed previous block node based on the difficulty retarget rules.
// This function differs from the exported CalcNextRequiredDifficulty in that
// the exported version uses the current best chain as the previous block node
// while this function accepts any block node.
func (client *TicketClient) GetNextRequiredDifficulty(block *types.Block, bits uint32) (uint32, error) {
	// Genesis block.
	if block == nil {
		return types.PowLimitBits, nil
	}
	blocksPerRetarget := int64(types.TargetTimespan / types.TargetTimePerBlock)
	// Return the previous block's difficulty requirements if this block
	// is not at a difficulty retarget interval.
	if (block.Height+1)%blocksPerRetarget != 0 {
		// For the main network (or any unrecognized networks), simply
		// return the previous block's difficulty requirements.
		return bits, nil
	}

	// Get the block node at the previous retarget (targetTimespan days
	// worth of blocks).
	firstBlock, err := client.RequestBlock(block.Height - blocksPerRetarget)
	if err != nil {
		return 0, err
	}
	if firstBlock == nil {
		return 0, types.ErrBlockNotFound
	}

	// Limit the amount of adjustment that can occur to the previous
	// difficulty.
	actualTimespan := block.BlockTime - firstBlock.BlockTime
	adjustedTimespan := actualTimespan
	targetTimespan := int64(types.TargetTimespan / time.Second)

	minRetargetTimespan := targetTimespan / (types.RetargetAdjustmentFactor)
	maxRetargetTimespan := targetTimespan * types.RetargetAdjustmentFactor
	if actualTimespan < int64(minRetargetTimespan) {
		adjustedTimespan = int64(minRetargetTimespan)
	} else if actualTimespan > maxRetargetTimespan {
		adjustedTimespan = maxRetargetTimespan
	}

	// Calculate new target difficulty as:
	//  currentDifficulty * (adjustedTimespan / targetTimespan)
	// The result uses integer division which means it will be slightly
	// rounded down.  Bitcoind also uses integer division to calculate this
	// result.
	oldTarget := CompactToBig(bits)
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(adjustedTimespan))
	targetTimeSpan := int64(types.TargetTimespan / time.Second)
	newTarget.Div(newTarget, big.NewInt(targetTimeSpan))

	// Limit new value to the proof of work limit.
	if newTarget.Cmp(types.PowLimit) > 0 {
		newTarget.Set(types.PowLimit)
	}

	// Log new target difficulty and return it.  The new target logging is
	// intentionally converting the bits back to a number instead of using
	// newTarget since conversion to the compact representation loses
	// precision.
	newTargetBits := BigToCompact(newTarget)
	slog.Info("Difficulty retarget at ", "block height %d", block.Height+1)
	slog.Info("Old target ", "%08x", bits, "(%064x)", oldTarget)
	slog.Info("New target ", "%08x", newTargetBits, "(%064x)", CompactToBig(newTargetBits))
	slog.Info("Timespan", "Actual timespan", time.Duration(actualTimespan)*time.Second,
		"adjusted timespan", time.Duration(adjustedTimespan)*time.Second,
		"target timespan", types.TargetTimespan)

	return newTargetBits, nil
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
