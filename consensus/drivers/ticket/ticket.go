package ticket

import (
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/difficulty"
	"gitlab.33.cn/chain33/chain33/consensus/drivers"
	driver "gitlab.33.cn/chain33/chain33/executor/drivers"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
)

var (
	tlog          = log15.New("module", "ticket")
	defaultModify = []byte("modify")
)

type Client struct {
	*drivers.BaseClient
	//ticket list for miner
	tlist    *types.ReplyTicketList
	privmap  map[string]crypto.PrivKey
	ticketmu sync.Mutex
	done     chan struct{}
}

func New(cfg *types.Consensus) *Client {
	c := drivers.NewBaseClient(cfg)
	t := &Client{c, &types.ReplyTicketList{}, nil, sync.Mutex{}, make(chan struct{})}
	c.SetChild(t)
	go t.flushTicketBackend()
	return t
}

func (client *Client) flushTicketBackend() {
	ticket := time.NewTicker(time.Hour)
	defer ticket.Stop()
Loop:
	for {
		select {
		case <-ticket.C:
			client.flushTicket()
		case <-client.done:
			break Loop
		}
	}
}

func (client *Client) Close() {
	close(client.done)
	client.BaseClient.Close()
	tlog.Info("consensus ticket closed")
}

func (client *Client) CreateGenesisTx() (ret []*types.Transaction) {
	//给ticket 合约打 3亿 个币
	//产生3w张初始化ticket
	if types.IsTestNet() {
		return client.CreateGenesisTxTestNet()
	}
	tx1 := createTicket("184wj4nsgVxKyz2NhM3Yb5RK5Ap6AFRFq2", "1FB8L3DykVF7Y78bRfUrRcMZwesKue7CyR", 3000, 0)
	ret = append(ret, tx1...)

	tx2 := createTicket("1M4ns1eGHdHak3SNc2UTQB75vnXyJQd91s", "1Lw6QLShKVbKM6QvMaCQwTh5Uhmy4644CG", 3000, 0)
	ret = append(ret, tx2...)

	tx3 := createTicket("19ozyoUGPAQ9spsFiz9CJfnUCFeszpaFuF", "1PSYYfCbtSeT1vJTvSKmQvhz8y6VhtddWi", 3000, 0)
	ret = append(ret, tx3...)

	tx4 := createTicket("1MoEnCDhXZ6Qv5fNDGYoW6MVEBTBK62HP2", "1BG9ZoKtgU5bhKLpcsrncZ6xdzFCgjrZud", 3000, 0)
	ret = append(ret, tx4...)

	tx5 := createTicket("1FjKcxY7vpmMH6iB5kxNYLvJkdkQXddfrp", "1G7s64AgX1ySDcUdSW5vDa8jTYQMnZktCd", 3000, 0)
	ret = append(ret, tx5...)

	tx6 := createTicket("12T8QfKbCRBhQdRfnAfFbUwdnH7TDTm4vx", "1FiDC6XWHLe7fDMhof8wJ3dty24f6aKKjK", 3000, 0)
	ret = append(ret, tx6...)

	tx7 := createTicket("1bgg6HwQretMiVcSWvayPRvVtwjyKfz1J", "1AMvuuQ7V7FPQ4hkvHQdgNWy8wVL4d4hmp", 3000, 0)
	ret = append(ret, tx7...)

	tx8 := createTicket("1EwkKd9iU1pL2ZwmRAC5RrBoqFD1aMrQ2", "1ExRRLoJXa8LzXdNxnJvBkVNZpVw3QWMi4", 3000, 0)
	ret = append(ret, tx8...)

	tx9 := createTicket("1HFUhgxarjC7JLru1FLEY6aJbQvCSL58CB", "1KNGHukhbBnbWWnMYxu1C7YMoCj45Z3amm", 3000, 0)
	ret = append(ret, tx9...)

	tx10 := createTicket("1C9M1RCv2e9b4GThN9ddBgyxAphqMgh5zq", "1AH9HRd4WBJ824h9PP1jYpvRZ4BSA4oN6Y", 4733, 0)
	ret = append(ret, tx10...)
	return
}

func (client *Client) CreateGenesisTxTestNet() (ret []*types.Transaction) {
	//给ticket 合约打 3亿 个币
	//产生3w张初始化ticket
	tx1 := createTicket("12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt", 10000, 0)
	ret = append(ret, tx1...)

	tx2 := createTicket("1PUiGcbsccfxW3zuvHXZBJfznziph5miAo", "1EbDHAXpoiewjPLX9uqoz38HsKqMXayZrF", 10000, 0)
	ret = append(ret, tx2...)

	tx3 := createTicket("1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX", "1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa", 10000, 0)
	ret = append(ret, tx3...)
	return
}

//316190000 coins
func createTicket(minerAddr, returnAddr string, count int32, height int64) (ret []*types.Transaction) {
	tx1 := types.Transaction{}
	tx1.Execer = []byte("coins")

	//给hotkey 10000 个币，作为miner的手续费
	tx1.To = minerAddr
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{Amount: types.GetP(height).TicketPrice}
	tx1.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx1)

	tx2 := types.Transaction{}

	tx2.Execer = []byte("coins")
	tx2.To = driver.ExecAddress("ticket")
	//gen payload
	g = &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{int64(count) * types.GetP(height).TicketPrice, returnAddr}
	tx2.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx2)

	tx3 := types.Transaction{}
	tx3.Execer = []byte("ticket")
	tx3.To = driver.ExecAddress("ticket")
	gticket := &types.TicketAction_Genesis{}
	gticket.Genesis = &types.TicketGenesis{minerAddr, returnAddr, count}
	tx3.Payload = types.Encode(&types.TicketAction{Value: gticket, Ty: types.TicketActionGenesis})
	ret = append(ret, &tx3)
	return ret
}

func (client *Client) ProcEvent(msg queue.Message) bool {
	if msg.Ty == types.EventFlushTicket {
		client.flushTicketMsg(msg)
	} else if msg.Ty == types.EventGetTicketCount {
		client.getTicketCountMsg(msg)
	} else {
		msg.ReplyErr("Client", types.ErrActionNotSupport)
	}
	return true
}

func (client *Client) privFromBytes(privkey []byte) (crypto.PrivKey, error) {
	cr, err := crypto.New(types.GetSignatureTypeName(types.SECP256K1))
	if err != nil {
		return nil, err
	}
	return cr.PrivKeyFromBytes(privkey)
}

func (client *Client) getTickets() ([]*types.Ticket, []crypto.PrivKey, error) {
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
	tlog.Info("getTickets", "ticket n", len(reply.Tickets), "nkey", len(keys))
	return reply.Tickets, keys, nil
}

func (client *Client) getTicketCount() int64 {
	client.ticketmu.Lock()
	defer client.ticketmu.Unlock()
	return int64(len(client.tlist.Tickets))
}

func (client *Client) getTicketCountMsg(msg queue.Message) {
	//list accounts
	var ret types.Int64
	ret.Data = client.getTicketCount()
	msg.Reply(client.GetQueueClient().NewMessage("", types.EventReplyGetTicketCount, &ret))
}

func (client *Client) setTicket(tlist *types.ReplyTicketList, privmap map[string]crypto.PrivKey) {
	client.ticketmu.Lock()
	defer client.ticketmu.Unlock()
	client.tlist = tlist
	client.privmap = privmap
	tlog.Debug("setTicket", "n", len(tlist.GetTickets()))
}

func (client *Client) flushTicket() error {
	//list accounts
	tickets, privs, err := client.getTickets()
	if err == types.ErrMinerNotStared {
		tlog.Error("flushTicket error", "err", "wallet miner not start")
		client.setTicket(nil, nil)
		return nil
	}
	if err != nil && err != types.ErrMinerNotStared {
		tlog.Error("flushTicket error", "err", err)
		return err
	}
	client.setTicket(&types.ReplyTicketList{tickets}, getPrivMap(privs))
	return nil
}

func (client *Client) flushTicketMsg(msg queue.Message) {
	//list accounts
	err := client.flushTicket()
	msg.ReplyErr("FlushTicket", err)
}

func getPrivMap(privs []crypto.PrivKey) map[string]crypto.PrivKey {
	list := make(map[string]crypto.PrivKey)
	for _, priv := range privs {
		addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
		list[addr] = priv
	}
	return list
}

func (client *Client) getMinerTx(current *types.Block) (*types.TicketAction, error) {
	//检查第一个笔交易的execs, 以及执行状态
	if len(current.Txs) == 0 {
		return nil, types.ErrEmptyTx
	}
	baseTx := current.Txs[0]
	//判断交易类型和执行情况
	var ticketAction types.TicketAction
	err := types.Decode(baseTx.GetPayload(), &ticketAction)
	if err != nil {
		return nil, err
	}
	if ticketAction.GetTy() != types.TicketActionMiner {
		return nil, types.ErrCoinBaseTxType
	}
	//判断交易执行是否OK
	if ticketAction.GetMiner() == nil {
		return nil, types.ErrEmptyMinerTx
	}
	return &ticketAction, nil
}

func (client *Client) getModify(block *types.Block) ([]byte, error) {
	ticketAction, err := client.getMinerTx(block)
	if err != nil {
		return defaultModify, err
	}
	return ticketAction.GetMiner().GetModify(), nil
}

func (client *Client) GetModify(beg, end int64) ([]byte, error) {
	//通过某个区间计算modify
	timeSource := int64(0)
	total := int64(0)
	newmodify := ""
	for i := beg; i <= end; i++ {
		block, err := client.RequestBlock(i)
		if err != nil {
			return defaultModify, err
		}
		timeSource += block.BlockTime
		if total == 0 {
			total = block.BlockTime
		}
		if timeSource%4 == 0 {
			total += block.BlockTime
		}
		if i == end {
			ticketAction, err := client.getMinerTx(block)
			if err != nil {
				return defaultModify, err
			}
			last := ticketAction.GetMiner().GetModify()
			newmodify = fmt.Sprintf("%s:%d", string(last), total)
		}
	}
	modify := common.ToHex(common.Sha256([]byte(newmodify)))
	return []byte(modify), nil
}

func (client *Client) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	cfg := types.GetP(current.Block.Height)
	if current.Block.BlockTime-types.Now().Unix() > cfg.FutureBlockTime {
		return types.ErrFutureBlock
	}
	ticketAction, err := client.getMinerTx(current.Block)
	if err != nil {
		return err
	}
	if parent.Height+1 != current.Block.Height {
		return types.ErrBlockHeight
	}
	//判断exec 是否成功
	if current.Receipts[0].Ty != types.ExecOk {
		return types.ErrCoinBaseExecErr
	}
	//check reward 的值是否正确
	miner := ticketAction.GetMiner()
	if miner.Reward != (cfg.CoinReward + calcTotalFee(current.Block)) {
		return types.ErrCoinbaseReward
	}
	if miner.Bits != current.Block.Difficulty {
		return types.ErrBlockHeaderDifficulty
	}
	//check modify:

	//通过判断区块的难度Difficulty
	//1. target >= currentdiff
	//2.  current bit == target
	target, modify, err := client.getNextTarget(parent, parent.Difficulty)
	if err != nil {
		return err
	}
	if string(modify) != string(miner.Modify) {
		return types.ErrModify
	}
	currentdiff := client.getCurrentTarget(current.Block.BlockTime, miner.TicketId, miner.Modify)
	if currentdiff.Sign() < 0 {
		return types.ErrCoinBaseTarget
	}
	//当前难度
	currentTarget := difficulty.CompactToBig(current.Block.Difficulty)
	if currentTarget.Cmp(difficulty.CompactToBig(miner.Bits)) != 0 {
		tlog.Error("block error: calc tagget not the same to miner",
			"cacl", printBInt(currentTarget), "current", printBInt(difficulty.CompactToBig(miner.Bits)))
		return types.ErrCoinBaseTarget
	}
	if currentTarget.Cmp(target) != 0 {
		tlog.Error("block error: calc tagget not the same to target",
			"cacl", printBInt(currentTarget), "current", printBInt(target))
		return types.ErrCoinBaseTarget
	}
	if currentdiff.Cmp(currentTarget) > 0 {
		tlog.Error("block error: diff not fit the tagget",
			"current", printBInt(currentdiff), "taget", printBInt(target))
		return types.ErrCoinBaseTarget
	}
	if current.Block.Size() > int(types.MaxBlockSize) {
		return types.ErrBlockSize
	}
	return nil
}

func (client *Client) getNextTarget(block *types.Block, bits uint32) (*big.Int, []byte, error) {
	if block.Height == 0 {
		powLimit := difficulty.CompactToBig(types.GetP(0).PowLimitBits)
		return powLimit, defaultModify, nil
	}
	targetBits, modify, err := client.GetNextRequiredDifficulty(block, bits)
	if err != nil {
		return nil, nil, err
	}
	return difficulty.CompactToBig(targetBits), modify, nil
}

func (client *Client) getCurrentTarget(blocktime int64, id string, modify []byte) *big.Int {
	s := fmt.Sprintf("%d:%s:%x", blocktime, id, modify)
	hash := common.Sha2Sum([]byte(s))
	num := difficulty.HashToBig(hash[:])
	return difficulty.CompactToBig(difficulty.BigToCompact(num))
}

// calcNextRequiredDifficulty calculates the required difficulty for the block
// after the passed previous block node based on the difficulty retarget rules.
// This function differs from the exported CalcNextRequiredDifficulty in that
// the exported version uses the current best chain as the previous block node
// while this function accepts any block node.
func (client *Client) GetNextRequiredDifficulty(block *types.Block, bits uint32) (uint32, []byte, error) {
	// Genesis block.
	if block == nil {
		return types.GetP(0).PowLimitBits, defaultModify, nil
	}
	cfg := types.GetP(block.Height)
	blocksPerRetarget := int64(cfg.TargetTimespan / cfg.TargetTimePerBlock)
	// Return the previous block's difficulty requirements if this block
	// is not at a difficulty retarget interval.
	if (block.Height+1) <= blocksPerRetarget || (block.Height+1)%blocksPerRetarget != 0 {
		// For the main network (or any unrecognized networks), simply
		// return the previous block's difficulty requirements.
		modify, err := client.getModify(block)
		if err != nil {
			return bits, defaultModify, err
		}
		return bits, modify, nil
	}

	// Get the block node at the previous retarget (targetTimespan days
	// worth of blocks).
	firstBlock, err := client.RequestBlock(block.Height + 1 - blocksPerRetarget)
	if err != nil {
		return cfg.PowLimitBits, defaultModify, err
	}
	if firstBlock == nil {
		return cfg.PowLimitBits, defaultModify, types.ErrBlockNotFound
	}

	modify, err := client.GetModify(block.Height+1-blocksPerRetarget, block.Height)
	if err != nil {
		return cfg.PowLimitBits, defaultModify, err
	}
	// Limit the amount of adjustment that can occur to the previous
	// difficulty.
	actualTimespan := block.BlockTime - firstBlock.BlockTime
	adjustedTimespan := actualTimespan
	targetTimespan := int64(cfg.TargetTimespan / time.Second)

	minRetargetTimespan := targetTimespan / (cfg.RetargetAdjustmentFactor)
	maxRetargetTimespan := targetTimespan * cfg.RetargetAdjustmentFactor
	if actualTimespan < minRetargetTimespan {
		adjustedTimespan = minRetargetTimespan
	} else if actualTimespan > maxRetargetTimespan {
		adjustedTimespan = maxRetargetTimespan
	}

	// Calculate new target difficulty as:
	//  currentDifficulty * (adjustedTimespan / targetTimespan)
	// The result uses integer division which means it will be slightly
	// rounded down.  Bitcoind also uses integer division to calculate this
	// result.
	oldTarget := difficulty.CompactToBig(bits)
	newTarget := new(big.Int).Mul(oldTarget, big.NewInt(adjustedTimespan))
	newTarget.Div(newTarget, big.NewInt(targetTimespan))

	// Limit new value to the proof of work limit.
	powLimit := difficulty.CompactToBig(cfg.PowLimitBits)
	if newTarget.Cmp(powLimit) > 0 {
		newTarget.Set(powLimit)
	}

	// Log new target difficulty and return it.  The new target logging is
	// intentionally converting the bits back to a number instead of using
	// newTarget since conversion to the compact representation loses
	// precision.
	newTargetBits := difficulty.BigToCompact(newTarget)
	tlog.Info(fmt.Sprintf("Difficulty retarget at block height %d", block.Height+1))
	tlog.Info(fmt.Sprintf("Old target %08x, (%064x)", bits, oldTarget))
	tlog.Info(fmt.Sprintf("New target %08x, (%064x)", newTargetBits, difficulty.CompactToBig(newTargetBits)))
	tlog.Info("Timespan", "Actual timespan", time.Duration(actualTimespan)*time.Second,
		"adjusted timespan", time.Duration(adjustedTimespan)*time.Second,
		"target timespan", cfg.TargetTimespan)
	prevmodify, err := client.getModify(block)
	if err != nil {
		panic(err)
	}
	tlog.Info("UpdateModify", "prev", string(prevmodify), "current", string(modify))
	return newTargetBits, modify, nil
}

func printBInt(data *big.Int) string {
	txt := data.Text(16)
	return strings.Repeat("0", 64-len(txt)) + txt
}

func (client *Client) searchTargetTicket(parent, block *types.Block) (*types.Ticket, crypto.PrivKey, *big.Int, []byte, int, error) {
	bits := parent.Difficulty
	diff, modify, err := client.getNextTarget(parent, bits)
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}
	client.ticketmu.Lock()
	defer client.ticketmu.Unlock()
	for i := 0; i < len(client.tlist.Tickets); i++ {
		ticket := client.tlist.Tickets[i]
		if ticket == nil {
			continue
		}
		//已经到成熟器
		if !ticket.IsGenesis && block.BlockTime-ticket.CreateTime <= types.GetP(block.Height).TicketFrozenTime {
			continue
		}
		//find priv key
		priv := client.privmap[ticket.MinerAddress]
		currentdiff := client.getCurrentTarget(block.BlockTime, ticket.TicketId, modify)
		if currentdiff.Cmp(diff) >= 0 { //难度要大于前一个，注意数字越小难度越大
			continue
		}
		tlog.Info("currentdiff", "hex", printBInt(currentdiff))
		tlog.Info("FindBlock", "height------->", block.Height, "ntx", len(block.Txs))
		return ticket, priv, diff, modify, i, nil
	}
	return nil, nil, nil, nil, 0, nil
}

func (client *Client) delTicket(ticket *types.Ticket, index int) {
	client.ticketmu.Lock()
	defer client.ticketmu.Unlock()
	//1. 结构体没有被重新调整过
	oldticket := client.tlist.Tickets[index]
	if oldticket.TicketId == ticket.TicketId {
		client.tlist.Tickets[index] = nil
	}
	//2. 全表search
	for i := 0; i < len(client.tlist.Tickets); i++ {
		oldticket = client.tlist.Tickets[i]
		if oldticket == nil {
			continue
		}
		if oldticket.TicketId == ticket.TicketId {
			client.tlist.Tickets[i] = nil
			return
		}
	}
}

func (client *Client) Miner(parent, block *types.Block) bool {
	//add miner address
	ticket, priv, diff, modify, index, err := client.searchTargetTicket(parent, block)
	if err != nil {
		tlog.Error("Miner", "err", err)
		newblock, err := client.RequestLastBlock()
		if err != nil {
			tlog.Error("Miner.RequestLastBlock", "err", err)
		}
		client.SetCurrentBlock(newblock)
		return false
	}
	if ticket == nil {
		return false
	}
	client.addMinerTx(parent, block, diff, priv, ticket.TicketId, modify)
	err = client.WriteBlock(parent.StateHash, block)
	if err != nil {
		return false
	}
	client.delTicket(ticket, index)
	return true
}

//gas 直接燃烧
func calcTotalFee(block *types.Block) (total int64) {
	return 0
}

func (client *Client) addMinerTx(parent, block *types.Block, diff *big.Int, priv crypto.PrivKey, tid string, modify []byte) {
	//return 0 always
	fee := calcTotalFee(block)

	var ticketAction types.TicketAction
	miner := &types.TicketMiner{}
	miner.TicketId = tid
	miner.Bits = difficulty.BigToCompact(diff)
	miner.Modify = modify
	miner.Reward = types.GetP(block.Height).CoinReward + fee
	ticketAction.Value = &types.TicketAction_Miner{miner}
	ticketAction.Ty = types.TicketActionMiner
	//构造transaction
	tx := client.createMinerTx(&ticketAction, priv)
	//unshift
	block.Difficulty = miner.Bits
	//判断是替换还是append
	_, err := client.getMinerTx(block)
	if err != nil {
		block.Txs = append([]*types.Transaction{tx}, block.Txs...)
	} else {
		//ticket miner 交易已经存在
		block.Txs[0] = tx
	}
}

func (client *Client) createMinerTx(ticketAction proto.Message, priv crypto.PrivKey) *types.Transaction {
	tx := &types.Transaction{}
	tx.Execer = []byte("ticket")
	tx.Fee = types.MinFee
	tx.Nonce = client.RandInt64()
	tx.To = address.ExecAddress("ticket")
	tx.Payload = types.Encode(ticketAction)
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func (client *Client) createBlock() (*types.Block, *types.Block) {
	lastBlock := client.GetCurrentBlock()
	var newblock types.Block
	newblock.ParentHash = lastBlock.Hash()
	newblock.Height = lastBlock.Height + 1
	newblock.BlockTime = types.Now().Unix()
	if lastBlock.BlockTime >= newblock.BlockTime {
		newblock.BlockTime = lastBlock.BlockTime + 1
	}
	txs := client.RequestTx(int(types.GetP(newblock.Height).MaxTxNumber)-1, nil)
	client.AddTxsToBlock(&newblock, txs)
	return &newblock, lastBlock
}

func (client *Client) updateBlock(newblock *types.Block, txHashList [][]byte) (*types.Block, [][]byte) {
	lastBlock := client.GetCurrentBlock()
	//需要去重复tx
	if lastBlock.Height != newblock.Height-1 {
		newblock.Txs = client.CheckTxDup(newblock.Txs)
	}
	newblock.ParentHash = lastBlock.Hash()
	newblock.Height = lastBlock.Height + 1
	newblock.BlockTime = types.Now().Unix()
	cfg := types.GetP(newblock.Height)
	var txs []*types.Transaction
	if len(newblock.Txs) < int(cfg.MaxTxNumber-1) {
		txs = client.RequestTx(int(cfg.MaxTxNumber)-1-len(newblock.Txs), txHashList)
	}
	//tx 有更新
	if len(txs) > 0 {
		//防止区块过大
		txs = client.AddTxsToBlock(newblock, txs)
		if len(txs) > 0 {
			txHashList = append(txHashList, getTxHashes(txs)...)
		}
	}
	if lastBlock.BlockTime >= newblock.BlockTime {
		newblock.BlockTime = lastBlock.BlockTime + 1
	}
	return lastBlock, txHashList
}

func (client *Client) CreateBlock() {
	for {
		if !client.IsMining() || !(client.IsCaughtUp() || client.Cfg.GetForceMining()) {
			tlog.Debug("createblock.ismining is disable or client is caughtup is false")
			time.Sleep(time.Second)
			continue
		}
		if client.getTicketCount() == 0 {
			tlog.Debug("createblock.getticketcount = 0")
			time.Sleep(time.Second)
			continue
		}
		block, lastBlock := client.createBlock()
		hashlist := getTxHashes(block.Txs)
		for !client.Miner(lastBlock, block) {
			//加入新的txs, 继续挖矿
			lasttime := block.BlockTime
			//只有时间增加了1s影响，影响难度计算了，才会去更新区块
			for lasttime >= types.Now().Unix() {
				time.Sleep(time.Second / 10)
			}
			lastBlock, hashlist = client.updateBlock(block, hashlist)
		}
	}
}

func getTxHashes(txs []*types.Transaction) (hashes [][]byte) {
	hashes = make([][]byte, len(txs))
	for i := 0; i < len(txs); i++ {
		hashes[i] = txs[i].Hash()
	}
	return hashes
}

func (client *Client) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, []*types.Transaction, error) {
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
	//判断txs[0] 是否执行OK
	if block.Height > 0 {
		err = client.CheckBlock(client.GetCurrentBlock(), blockdetail)
		if err != nil {
			return nil, deltx, err
		}
	}
	return blockdetail, deltx, nil
}
