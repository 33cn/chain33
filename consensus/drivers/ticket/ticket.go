package ticket

import (
	"errors"
	"fmt"
	"math/big"
	"strings"
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
	"github.com/inconshreveable/log15"
)

var (
	tlog          = log15.New("module", "ticket")
	powLimit      = common.CompactToBig(types.PowLimitBits)
	defaultModify = []byte("modify")
)

type TicketClient struct {
	*drivers.BaseClient
	//ticket list for miner
	tlist    *types.ReplyTicketList
	privmap  map[string]crypto.PrivKey
	ticketmu sync.Mutex
	done     chan struct{}
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
	client.BaseClient.Close()
	tlog.Info("consensus ticket closed")
}

func (client *TicketClient) CreateGenesisTx() (ret []*types.Transaction) {
	//给ticket 合约打 3亿 个币
	//产生3w张初始化ticket
	tx1 := createTicket("12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv", "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt", 10000)
	ret = append(ret, tx1...)

	tx2 := createTicket("1PUiGcbsccfxW3zuvHXZBJfznziph5miAo", "1EbDHAXpoiewjPLX9uqoz38HsKqMXayZrF", 10000)
	ret = append(ret, tx2...)

	tx3 := createTicket("1EDnnePAZN48aC2hiTDzhkczfF39g1pZZX", "1KcCVZLSQYRUwE5EXTsAoQs9LuJW6xwfQa", 10000)
	ret = append(ret, tx3...)
	return
}

func createTicket(minerAddr, returnAddr string, count int32) (ret []*types.Transaction) {
	tx1 := types.Transaction{}
	tx1.Execer = []byte("coins")

	//给hotkey 10000 个币，作为miner的手续费
	tx1.To = minerAddr
	//gen payload
	g := &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{Amount: types.TicketPrice}
	tx1.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx1)

	tx2 := types.Transaction{}

	tx2.Execer = []byte("coins")
	tx2.To = execdrivers.ExecAddress("ticket").String()
	//gen payload
	g = &types.CoinsAction_Genesis{}
	g.Genesis = &types.CoinsGenesis{int64(count) * types.TicketPrice, returnAddr}
	tx2.Payload = types.Encode(&types.CoinsAction{Value: g, Ty: types.CoinsActionGenesis})
	ret = append(ret, &tx2)

	tx3 := types.Transaction{}
	tx3.Execer = []byte("ticket")
	tx3.To = execdrivers.ExecAddress("ticket").String()
	gticket := &types.TicketAction_Genesis{}
	gticket.Genesis = &types.TicketGenesis{minerAddr, returnAddr, count}
	tx3.Payload = types.Encode(&types.TicketAction{Value: gticket, Ty: types.TicketActionGenesis})
	ret = append(ret, &tx3)
	return ret
}

func (client *TicketClient) ProcEvent(msg queue.Message) {
	if msg.Ty == types.EventFlushTicket {
		client.flushTicketMsg(msg)
	} else if msg.Ty == types.EventGetTicketCount {
		client.getTicketCountMsg(msg)
	} else {
		msg.ReplyErr("TicketClient", types.ErrActionNotSupport)
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
	tlog.Info("getTickets", "ticket n", len(reply.Tickets), "nkey", len(keys))
	return reply.Tickets, keys, nil
}

func (client *TicketClient) getTicketCount() int64 {
	client.ticketmu.Lock()
	defer client.ticketmu.Unlock()
	return int64(len(client.tlist.Tickets))
}

func (client *TicketClient) getTicketCountMsg(msg queue.Message) {
	//list accounts
	var ret types.Int64
	ret.Data = client.getTicketCount()
	msg.Reply(client.GetQueueClient().NewMessage("", types.EventReplyGetTicketCount, &ret))
}

func (client *TicketClient) setTicket(tlist *types.ReplyTicketList, privmap map[string]crypto.PrivKey) {
	client.ticketmu.Lock()
	defer client.ticketmu.Unlock()
	client.tlist = tlist
	client.privmap = privmap
}

func (client *TicketClient) flushTicket() error {
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

func (client *TicketClient) getMinerTx(current *types.Block) (*types.TicketAction, error) {
	//检查第一个笔交易的execs, 以及执行状态
	if len(current.Txs) == 0 {
		return nil, types.ErrEmptyTx
	}
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

func (client *TicketClient) getModify(block *types.Block) ([]byte, error) {
	ticketAction, err := client.getMinerTx(block)
	if err != nil {
		return defaultModify, err
	}
	return ticketAction.GetMiner().GetModify(), nil
}

func (client *TicketClient) GetModify(beg, end int64) ([]byte, error) {
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

func (client *TicketClient) CheckBlock(parent *types.Block, current *types.BlockDetail) error {
	if current.Block.BlockTime-time.Now().Unix() > types.FutureBlockTime {
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
	if miner.Reward != (types.CoinReward + calcTotalFee(current.Block)) {
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
	currentTarget := common.CompactToBig(current.Block.Difficulty)
	if currentTarget.Cmp(common.CompactToBig(miner.Bits)) != 0 {
		tlog.Error("block error: calc tagget not the same to miner",
			"cacl", printBInt(currentTarget), "current", printBInt(common.CompactToBig(miner.Bits)))
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
	return nil
}

func (client *TicketClient) getNextTarget(block *types.Block, bits uint32) (*big.Int, []byte, error) {
	if block.Height == 0 {
		return powLimit, defaultModify, nil
	}
	targetBits, modify, err := client.GetNextRequiredDifficulty(block, bits)
	if err != nil {
		return nil, nil, err
	}
	return common.CompactToBig(targetBits), modify, nil
}

func (client *TicketClient) getCurrentTarget(blocktime int64, id string, modify []byte) *big.Int {
	s := fmt.Sprintf("%d:%s:%x", blocktime, id, modify)
	hash := common.Sha2Sum([]byte(s))
	num := common.HashToBig(hash[:])
	return common.CompactToBig(common.BigToCompact(num))
}

// calcNextRequiredDifficulty calculates the required difficulty for the block
// after the passed previous block node based on the difficulty retarget rules.
// This function differs from the exported CalcNextRequiredDifficulty in that
// the exported version uses the current best chain as the previous block node
// while this function accepts any block node.
func (client *TicketClient) GetNextRequiredDifficulty(block *types.Block, bits uint32) (uint32, []byte, error) {
	// Genesis block.
	if block == nil {
		return types.PowLimitBits, defaultModify, nil
	}
	blocksPerRetarget := int64(types.TargetTimespan / types.TargetTimePerBlock)
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
		return types.PowLimitBits, defaultModify, err
	}
	if firstBlock == nil {
		return types.PowLimitBits, defaultModify, types.ErrBlockNotFound
	}

	modify, err := client.GetModify(block.Height+1-blocksPerRetarget, block.Height)
	if err != nil {
		return types.PowLimitBits, defaultModify, err
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
	if newTarget.Cmp(powLimit) > 0 {
		newTarget.Set(powLimit)
	}

	// Log new target difficulty and return it.  The new target logging is
	// intentionally converting the bits back to a number instead of using
	// newTarget since conversion to the compact representation loses
	// precision.
	newTargetBits := common.BigToCompact(newTarget)
	tlog.Info(fmt.Sprintf("Difficulty retarget at block height %d", block.Height+1))
	tlog.Info(fmt.Sprintf("Old target %08x, (%064x)", bits, oldTarget))
	tlog.Info(fmt.Sprintf("New target %08x, (%064x)", newTargetBits, common.CompactToBig(newTargetBits)))
	tlog.Info("Timespan", "Actual timespan", time.Duration(actualTimespan)*time.Second,
		"adjusted timespan", time.Duration(adjustedTimespan)*time.Second,
		"target timespan", types.TargetTimespan)
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

func (client *TicketClient) searchTargetTicket(parent, block *types.Block) (*types.Ticket, crypto.PrivKey, *big.Int, []byte, int, error) {
	bits := parent.Difficulty
	diff, modify, err := client.getNextTarget(parent, bits)
	if err != nil {
		return nil, nil, nil, nil, 0, err
	}
	tlog.Debug("target", "hex", printBInt(diff))
	client.ticketmu.Lock()
	defer client.ticketmu.Unlock()
	for i := 0; i < len(client.tlist.Tickets); i++ {
		ticket := client.tlist.Tickets[i]
		if ticket == nil {
			continue
		}
		//已经到成熟器
		if !ticket.IsGenesis && block.BlockTime-ticket.CreateTime <= types.TicketFrozenTime {
			continue
		}
		//find priv key
		priv := client.privmap[ticket.MinerAddress]
		currentdiff := client.getCurrentTarget(block.BlockTime, ticket.TicketId, modify)
		if currentdiff.Cmp(diff) >= 0 { //难度要大于前一个，注意数字越小难度越大
			continue
		}
		tlog.Info("currentdiff", "hex", printBInt(currentdiff))
		tlog.Info("ExecBlock", "height------->", block.Height, "ntx", len(block.Txs))
		return ticket, priv, diff, modify, i, nil
	}
	return nil, nil, nil, nil, 0, nil
}

func (client *TicketClient) delTicket(ticket *types.Ticket, index int) {
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

func (client *TicketClient) Miner(parent, block *types.Block) bool {
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
	beg := time.Now()
	receipts, err := client.addMinerTx(parent, block, diff, priv, ticket.TicketId, modify)
	if err != nil {
		return false
	}
	block.TxHash = merkle.CalcMerkleRoot(block.Txs)
	blockdetail := &types.BlockDetail{block, getReceiptsData(receipts)}
	if err := client.CheckBlock(parent, blockdetail); err != nil {
		return false
	}
	err = client.saveBlock(parent, block, receipts)
	if err != nil {
		return false
	}
	tlog.Info("ExecBlock", "cost", time.Now().Sub(beg))
	err = client.WriteBlock(blockdetail)
	if err != nil {
		return false
	}
	client.delTicket(ticket, index)
	return true
}

func (client *TicketClient) saveBlock(parent, block *types.Block, receipts *types.Receipts) error {
	var maplist = make(map[string]*types.KeyValue)
	var kvset []*types.KeyValue
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		kvs := receipt.KV
		for _, kv := range kvs {
			if item, ok := maplist[string(kv.Key)]; ok {
				item.Value = kv.Value //更新item 的value
			} else {
				maplist[string(kv.Key)] = kv
				kvset = append(kvset, kv)
			}
		}
	}
	if kvset == nil {
		block.StateHash = parent.StateHash
	} else {
		block.StateHash = util.ExecKVMemSet(client.GetQueue(), parent.StateHash, kvset)
	}
	return util.ExecKVSetCommit(client.GetQueue(), block.StateHash)
}

func getReceiptsData(receipts *types.Receipts) (datas []*types.ReceiptData) {
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		datas = append(datas, &types.ReceiptData{receipt.Ty, receipt.Logs})
	}
	return datas
}

//gas 直接燃烧
func calcTotalFee(block *types.Block) (total int64) {
	return 0
}

func (client *TicketClient) addMinerTx(parent, block *types.Block, diff *big.Int, priv crypto.PrivKey, tid string, modify []byte) (*types.Receipts, error) {
	fee := calcTotalFee(block)
	var ticketAction types.TicketAction
	miner := &types.TicketMiner{}
	miner.TicketId = tid
	miner.Bits = common.BigToCompact(diff)
	miner.Modify = modify
	miner.Reward = types.CoinReward + fee
	ticketAction.Value = &types.TicketAction_Miner{miner}
	ticketAction.Ty = types.TicketActionMiner
	//构造transaction
	tx := client.createMinerTx(&ticketAction, priv)
	//unshift
	block.Difficulty = miner.Bits
	block.Txs = append([]*types.Transaction{tx}, block.Txs...)
	return client.execBlock(parent.StateHash, block, priv)
}

func (client *TicketClient) createMinerTx(ticketAction *types.TicketAction, priv crypto.PrivKey) *types.Transaction {
	tx := &types.Transaction{}
	tx.Execer = []byte("ticket")
	tx.Fee = types.MinFee
	tx.Nonce = client.RandInt64()
	tx.To = account.ExecAddress("ticket").String()
	tx.Payload = types.Encode(ticketAction)
	tx.Sign(types.SECP256K1, priv)
	return tx
}

func (client *TicketClient) createBlock() (*types.Block, *types.Block) {
	txs := client.RequestTx(int(types.MaxTxNumber) - 1)
	//check dup
	if len(txs) > 0 {
		//txs = client.CheckTxDup(txs)
	}
	lastBlock := client.GetCurrentBlock()
	var newblock types.Block
	newblock.ParentHash = lastBlock.Hash()
	newblock.Height = lastBlock.Height + 1
	newblock.Txs = txs
	newblock.BlockTime = time.Now().Unix()
	if lastBlock.BlockTime >= newblock.BlockTime {
		newblock.BlockTime = lastBlock.BlockTime + 1
	}
	return &newblock, lastBlock
}

func (client *TicketClient) updateBlock(newblock *types.Block) *types.Block {

	var txs []*types.Transaction
	//check dup
	if len(newblock.Txs) < int(types.MaxTxNumber-1) {
		txs = client.RequestTx(int(types.MaxTxNumber) - 1 - len(newblock.Txs))
		if len(txs) > 0 {
			txs = client.CheckTxDup(txs)
		}
	}
	lastBlock := client.GetCurrentBlock()
	newblock.ParentHash = lastBlock.Hash()
	newblock.Height = lastBlock.Height + 1
	if len(txs) > 0 {
		newblock.Txs = append(newblock.Txs, txs...)
	}
	newblock.BlockTime = time.Now().Unix()
	if lastBlock.BlockTime >= newblock.BlockTime {
		newblock.BlockTime = lastBlock.BlockTime + 1
	}
	return lastBlock
}

func (client *TicketClient) CreateBlock() {
	for {
		if !client.IsMining() {
			time.Sleep(time.Second)
			continue
		}
		if client.getTicketCount() == 0 {
			time.Sleep(time.Second)
			continue
		}
		block, lastBlock := client.createBlock()
		for !client.Miner(lastBlock, block) {
			time.Sleep(time.Second)
			//加入新的txs, 继续挖矿
			lastBlock = client.updateBlock(block)
		}
	}
}

func (client *TicketClient) execBlock(prevHash []byte, block *types.Block, priv crypto.PrivKey) (*types.Receipts, error) {
	//执行整个交易
	receipts := util.ExecTxList(client.GetQueue(), prevHash, block.Txs, block.GetHeader())
	var deltxlist = make(map[int]bool)
	var haserr bool
	for i := 0; i < len(receipts.Receipts); i++ {
		receipt := receipts.Receipts[i]
		if receipt.Ty == types.ExecErr {
			tlog.Error("exec tx err", "err", receipt)
			deltxlist[i] = true
			haserr = true
		}
	}
	if !haserr {
		return receipts, nil
	}
	//删除错误的交易
	if len(deltxlist) > 0 {
		var newtx []*types.Transaction
		var newreceipts []*types.Receipt
		for i := 0; i < len(block.Txs); i++ {
			if !deltxlist[i] {
				newtx = append(newtx, block.Txs[i])
				newreceipts = append(newreceipts, receipts.Receipts[i])
			}
		}
		block.Txs = newtx
		receipts.Receipts = newreceipts
	}
	//fee 直接燃烧，消失
	//比较好做的是，直接在block header 里面加一个 reward 字段
	//但是目前直接燃烧比较简单
	return receipts, nil
}

func (client *TicketClient) WriteBlock(blockdetail *types.BlockDetail) error {
	msg := client.GetQueueClient().NewMessage("blockchain", types.EventAddBlockDetail, blockdetail)
	client.GetQueueClient().Send(msg, true)
	resp, err := client.GetQueueClient().Wait(msg)
	if err != nil {
		return err
	}
	if resp.GetData().(*types.Reply).IsOk {
		client.SetCurrentBlock(blockdetail.Block)
	} else {
		//TODO:
		//把txs写回mempool
		reply := resp.GetData().(*types.Reply)
		return errors.New(string(reply.GetMsg()))
	}
	return nil
}

func (client *TicketClient) ExecBlock(prevHash []byte, block *types.Block) (*types.BlockDetail, error) {
	if block.Height > 0 {
		//在miner 中处理了
		return nil, nil
	}
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
	//判断txs[0] 是否执行OK
	if block.Height > 0 {
		err = client.CheckBlock(client.GetCurrentBlock(), blockdetail)
		if err != nil { //never happen
			return nil, err
		}
	}
	return blockdetail, nil
}
