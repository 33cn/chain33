package ticket

import (
	"fmt"
	"math/big"
	"sync"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/common/merkle"
	"code.aliyun.com/chain33/chain33/consensus/drivers"
	"code.aliyun.com/chain33/chain33/execs/execdrivers"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
	"code.aliyun.com/chain33/chain33/util"
	log "github.com/inconshreveable/log15"
)

var slog = log.New("module", "ticket")

type TicketClient struct {
	*drivers.BaseClient
	//ticket list for miner
	tlist   *types.ReplyTicketList
	privmap map[string]crypto.PrivKey
	mu      sync.Mutex
	done    chan struct{}
}

func New(cfg *types.Consensus) *TicketClient {
	c := drivers.NewBaseClient(cfg)
	t := &TicketClient{c, &types.ReplyTicketList{}, nil, sync.Mutex{}, make(chan struct{})}
	c.SetChild(t)
	go t.flushTicketBackend()
	return t
}

func (client *TicketClient) flushTicketBackend() {
	ticket := time.NewTicker(time.Hour)
	defer ticket.Stop()
	for {
		select {
		case <-ticket.C:
			client.flushTicket()
		case <-client.done:
			break
		}
	}
}

func (client *TicketClient) Close() {
	close(client.done)
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
	gticket.Genesis = &types.TicketGenesis{client.Cfg.HotkeyAddr, client.Cfg.Genesis, 30000}

	tx3.Payload = types.Encode(&types.TicketAction{Value: gticket, Ty: types.TicketActionGenesis})
	ret = append(ret, &tx3)
	return
}

func (client *TicketClient) ProcEvent(msg queue.Message) {
	if msg.Ty == types.EventFlushTicket {
		client.flushTicketMsg(msg)
	}
}

func (client *TicketClient) privFromBytes(privkey []byte) (crypto.PrivKey, error) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		return nil, err
	}
	return cr.PrivKeyFromBytes(privkey)
}

func (client *TicketClient) getTickets() ([]*types.Ticket, []crypto.PrivKey, error) {
	msg := client.GetQueueClient().NewMessage("wallet", types.EventWalletGetTickets, nil)
	client.GetQueueClient().Send(msg, true)
	resp, err := client.GetQueueClient().Wait(msg)
	if err != nil {
		return nil, nil, err
	}
	reply := resp.GetData().(types.Message).(*types.ReplyWalletTickets)
	var keys []crypto.PrivKey
	for i := 0; i < len(reply.Privkeys); i++ {
		priv, err := client.privFromBytes(reply.Privkeys[i])
		if err != nil {
			return nil, nil, err
		}
		keys = append(keys, priv)
	}
	return reply.Tickets, keys, nil
}

func (client *TicketClient) flushTicket() error {
	//list accounts
	tickets, privs, err := client.getTickets()
	if err != nil {
		return err
	}
	client.mu.Lock()
	client.privmap = getPrivMap(privs)
	client.tlist.Tickets = tickets
	client.mu.Unlock()
	return nil
}
func (client *TicketClient) flushTicketMsg(msg queue.Message) {
	//list accounts
	err := client.flushTicket()
	msg.ReplyErr("FlushTicket", err)
}

func getPrivMap(privs []crypto.PrivKey) map[string]crypto.PrivKey {
	list := make(map[string]crypto.PrivKey)
	for _, priv := range privs {
		addr := account.PubKeyToAddress(priv.PubKey().Bytes()).String()
		list[addr] = priv
	}
	return list
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
	if miner.Bits != current.Block.Difficulty {
		return types.ErrBlockHeaderDifficulty
	}
	//通过判断区块的难度Difficulty
	bits := parent.Difficulty
	diff := client.getNextTarget(parent, bits)
	currentdiff := client.getCurrentTarget(current.Block.BlockTime, miner.TicketId, miner.Modify)
	if currentdiff.Cmp(common.CompactToBig(miner.Bits)) != 0 {
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
	return common.CompactToBig(targetBits)
}

func (client *TicketClient) getCurrentTarget(blocktime int64, id string, modify []byte) *big.Int {
	s := fmt.Sprint("%d:%s:%x", blocktime, id, modify)
	hash := common.Sha2Sum([]byte(s))
	return common.HashToBig(hash[:])
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
	oldTarget := common.CompactToBig(bits)
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
	newTargetBits := common.BigToCompact(newTarget)
	slog.Info("Difficulty retarget at ", "block height %d", block.Height+1)
	slog.Info("Old target ", "%08x", bits, "(%064x)", oldTarget)
	slog.Info("New target ", "%08x", newTargetBits, "(%064x)", common.CompactToBig(newTargetBits))
	slog.Info("Timespan", "Actual timespan", time.Duration(actualTimespan)*time.Second,
		"adjusted timespan", time.Duration(adjustedTimespan)*time.Second,
		"target timespan", types.TargetTimespan)

	return newTargetBits, nil
}

func (client *TicketClient) Miner(block *types.Block) bool {
	//add miner address
	parent := client.GetCurrentBlock()
	bits := parent.Difficulty
	diff := client.getNextTarget(parent, bits)
	client.mu.Lock()
	defer client.mu.Unlock()
	for i := 0; i < len(client.tlist.Tickets); i++ {
		ticket := client.tlist.Tickets[i]
		//已经到成熟器
		if !ticket.IsGenesis && block.BlockTime-ticket.CreateTime <= 10*86400 {
			continue
		}
		//find priv key
		priv := client.privmap[ticket.MinerAddress]
		currentdiff := client.getCurrentTarget(block.BlockTime, ticket.TicketId, []byte("modify"))
		if currentdiff.Cmp(diff) > 0 { //难度要大于前一个，注意数字越小难度越大
			continue
		}
		var ticketAction types.TicketAction
		miner := &types.TicketMiner{}
		miner.TicketId = ticket.TicketId
		miner.Bits = common.BigToCompact(currentdiff)
		miner.Modify = []byte("modify")
		miner.Reward = types.CoinReward
		ticketAction.Value = &types.TicketAction_Miner{miner}
		//构造transaction
		tx := &types.Transaction{}
		tx.Execer = []byte("ticket")
		tx.Fee = types.MinFee
		tx.Nonce = client.RandInt64()
		tx.To = account.ExecAddress("ticket").String()
		tx.Payload = types.Encode(&ticketAction)
		tx.Sign(types.SECP256K1, priv)
		//unshift
		block.Difficulty = miner.Bits
		block.Txs = append([]*types.Transaction{tx}, block.Txs...)
		return true
	}
	return false
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
		newblock.BlockTime = time.Now().Unix()
		if lastBlock.BlockTime >= newblock.BlockTime {
			newblock.BlockTime = lastBlock.BlockTime + 1
		}
		//add miner tx
		if !client.Miner(&newblock) {
			continue
		}
		newblock.TxHash = merkle.CalcMerkleRoot(newblock.Txs)
		err := client.WriteBlock(lastBlock.StateHash, &newblock)
		if err != nil {
			issleep = true
			continue
		}
	}
}

func (client *TicketClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, error) {
	//exec block
	blockdetail, err := util.ExecBlock(client.GetQueue(), prevHash, block, false)
	if err != nil { //never happen
		return nil, err
	}
	if len(blockdetail.Block.Txs) == 0 {
		return nil, types.ErrNoTx
	}
	//判断txs[0] 是否执行OK
	if block.Height > 0 {
		err = client.CheckBlock(client.GetCurrentBlock(), blockdetail)
		if err != nil { //never happen
			return nil, err
		}
	}
	return blockdetail, nil
}
