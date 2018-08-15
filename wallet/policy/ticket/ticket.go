package ticket

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/client"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

var (
	minerAddrWhiteList = make(map[string]bool)

	bizlog = log15.New("module", "wallet.ticket")
)

func init() {
	wcom.RegisterPolicy(types.TicketX, New())
}

func New() wcom.WalletBizPolicy {
	return &ticketPolicy{}
}

type ticketPolicy struct {
	mtx            *sync.Mutex
	walletOperate  wcom.WalletOperate
	store          *ticketStore
	needFlush      bool
	miningTicket   *time.Ticker
	autoMinerFlag  int32
	isTicketLocked int32
	minertimeout   *time.Timer
}

func (policy *ticketPolicy) initFuncMap(walletOperate wcom.WalletOperate) {
	wcom.RegisterMsgFunc(types.EventCloseTickets, policy.onCloseTickets)
	wcom.RegisterMsgFunc(types.EventWalletGetTickets, policy.onWalletGetTickets)
	wcom.RegisterMsgFunc(types.EventWalletAutoMiner, policy.onWalletAutoMiner)
}

func (policy *ticketPolicy) setWalletOperate(walletBiz wcom.WalletOperate) {
	policy.mtx.Lock()
	defer policy.mtx.Unlock()
	policy.walletOperate = walletBiz
}

func (policy *ticketPolicy) getWalletOperate() wcom.WalletOperate {
	policy.mtx.Lock()
	defer policy.mtx.Unlock()
	return policy.walletOperate
}

func (policy *ticketPolicy) getAPI() client.QueueProtocolAPI {
	policy.mtx.Lock()
	defer policy.mtx.Unlock()
	return policy.walletOperate.GetAPI()
}

func (policy *ticketPolicy) IsAutoMining() bool {
	return policy.isAutoMining()
}

func (policy *ticketPolicy) IsTicketLocked() bool {
	return atomic.LoadInt32(&policy.isTicketLocked) != 0
}

func (policy *ticketPolicy) Init(walletBiz wcom.WalletOperate) {
	policy.mtx = &sync.Mutex{}
	policy.setWalletOperate(walletBiz)
	policy.store = NewStore(walletBiz.GetDBStore())
	policy.needFlush = false
	policy.isTicketLocked = 1
	policy.autoMinerFlag = policy.store.GetAutoMinerFlag()
	policy.miningTicket = time.NewTicker(2 * time.Minute)

	walletBiz.RegisterMineStatusReporter(policy)
	policy.initFuncMap(walletBiz)

	initMinerWhiteList(walletBiz.GetConfig())
	// 启动自动挖矿
	walletBiz.GetWaitGroup().Add(1)
	go policy.autoMining()
}

func (policy *ticketPolicy) OnClose() {
	policy.miningTicket.Stop()
}

func (policy *ticketPolicy) OnAddBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {
	receipt := block.Receipts[index]
	if policy.checkNeedFlushTicket(tx, receipt) {
		policy.needFlush = true
	}
}

func (policy *ticketPolicy) OnDeleteBlockTx(block *types.BlockDetail, tx *types.Transaction, index int32, dbbatch db.Batch) {

}

func (policy *ticketPolicy) SignTransaction(key crypto.PrivKey, req *types.ReqSignRawTx) (needSysSign bool, signtx string, err error) {
	needSysSign = true
	return
}

func (policy *ticketPolicy) OnWalletLocked() {
	// 钱包锁住时，不允许挖矿
	atomic.CompareAndSwapInt32(&policy.isTicketLocked, 0, 1)
}

//解锁超时处理，需要区分整个钱包的解锁或者只挖矿的解锁
func (policy *ticketPolicy) resetTimeout(Timeout int64) {
	if policy.minertimeout == nil {
		policy.minertimeout = time.AfterFunc(time.Second*time.Duration(Timeout), func() {
			//wallet.isTicketLocked = true
			atomic.CompareAndSwapInt32(&policy.isTicketLocked, 0, 1)
		})
	} else {
		policy.minertimeout.Reset(time.Second * time.Duration(Timeout))
	}
}

func (policy *ticketPolicy) OnWalletUnlocked(param *types.WalletUnLock) {
	if param.WalletOrTicket {
		atomic.CompareAndSwapInt32(&policy.isTicketLocked, 1, 0)
		if param.Timeout != 0 {
			policy.resetTimeout(param.Timeout)
		}
	}
	// 钱包解锁时，需要刷新，通知挖矿
	policy.flushTicket()
}

func (policy *ticketPolicy) OnCreateNewAccount(acc *types.Account) {
}

func (policy *ticketPolicy) OnImportPrivateKey(acc *types.Account) {
	// 账号导入时，需要刷新，通知挖矿
	policy.flushTicket()
}

func (policy *ticketPolicy) OnAddBlockFinish(block *types.BlockDetail) {
	if policy.needFlush {
		//policy.flushTicket()
	}
	policy.needFlush = false
}

func (policy *ticketPolicy) OnDeleteBlockFinish(block *types.BlockDetail) {
	if policy.needFlush {
		policy.flushTicket()
	}
	policy.needFlush = false
}

func (policy *ticketPolicy) flushTicket() {
	bizlog.Info("wallet FLUSH TICKET")
	api := policy.getAPI()
	api.Notify("consensus", types.EventFlushTicket, nil)
}

func (policy *ticketPolicy) needFlushTicket(tx *types.Transaction, receipt *types.ReceiptData) bool {
	pubkey := tx.Signature.GetPubkey()
	addr := address.PubKeyToAddress(pubkey)
	return policy.store.checkAddrIsInWallet(addr.String())
}

func (policy *ticketPolicy) checkNeedFlushTicket(tx *types.Transaction, receipt *types.ReceiptData) bool {
	if receipt.Ty != types.ExecOk {
		return false
	}
	return policy.needFlushTicket(tx, receipt)
}

func (policy *ticketPolicy) onCloseTickets(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyHashes)
	operater := policy.getWalletOperate()
	reply, err := policy.forceCloseTicket(operater.GetBlockHeight() + 1)
	if err != nil {
		bizlog.Error("onCloseTickets", "forceCloseTicket error", err.Error())
	} else {
		go func() {
			if len(reply.Hashes) > 0 {
				operater.WaitTxs(reply.Hashes)
				policy.flushTicket()
			}
		}()
	}
	return topic, retty, reply, err
}

func (policy *ticketPolicy) forceCloseTicket(height int64) (*types.ReplyHashes, error) {
	return policy.forceCloseAllTicket(height)
}

func (policy *ticketPolicy) forceCloseAllTicket(height int64) (*types.ReplyHashes, error) {
	keys, err := policy.getWalletOperate().GetAllPrivKeys()
	if err != nil {
		return nil, err
	}
	var hashes types.ReplyHashes
	for _, key := range keys {
		hash, err := policy.forceCloseTicketsByAddr(height, key)
		if err != nil {
			bizlog.Error("forceCloseAllTicket", "forceCloseTicketsByAddr error", err)
			continue
		}
		if hash == nil {
			continue
		}
		hashes.Hashes = append(hashes.Hashes, hash)
	}
	return &hashes, nil
}

func (policy *ticketPolicy) getTickets(addr string, status int32) ([]*types.Ticket, error) {
	reqaddr := &types.TicketList{addr, status}
	var req types.Query
	req.Execer = types.ExecerTicket
	req.FuncName = "TicketList"
	req.Payload = types.Encode(reqaddr)
	api := policy.getAPI()
	msg, err := api.Query(&req)
	if err != nil {
		bizlog.Error("getTickets", "Query error", err)
		return nil, err
	}
	reply := (*msg).(*types.ReplyTicketList)
	return reply.Tickets, nil
}

func (policy *ticketPolicy) forceCloseTicketsByAddr(height int64, priv crypto.PrivKey) ([]byte, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	tlist1, err1 := policy.getTickets(addr, 1)
	if err1 != nil && err1 != types.ErrNotFound {
		return nil, err1
	}
	tlist2, err2 := policy.getTickets(addr, 2)
	if err2 != nil && err2 != types.ErrNotFound {
		return nil, err1
	}
	tlist := append(tlist1, tlist2...)
	var ids []string
	var tl []*types.Ticket
	now := types.Now().Unix()
	for _, t := range tlist {
		if !t.IsGenesis {
			if t.Status == 1 && now-t.GetCreateTime() < types.GetP(height).TicketWithdrawTime {
				continue
			}
			if t.Status == 2 && now-t.GetCreateTime() < types.GetP(height).TicketWithdrawTime {
				continue
			}
			if t.Status == 2 && now-t.GetMinerTime() < types.GetP(height).TicketMinerWaitTime {
				continue
			}
		}
		tl = append(tl, t)
	}
	for i := 0; i < len(tl); i++ {
		ids = append(ids, tl[i].TicketId)
	}
	if len(ids) > 0 {
		return policy.closeTickets(priv, ids)
	}
	return nil, nil
}

//通过rpc 精选close 操作
func (policy *ticketPolicy) closeTickets(priv crypto.PrivKey, ids []string) ([]byte, error) {
	//每次最多close 200个
	end := 200
	if end > len(ids) {
		end = len(ids)
	}
	bizlog.Info("closeTickets", "ids", ids[0:end])
	ta := &types.TicketAction{}
	tclose := &types.TicketClose{ids[0:end]}
	ta.Value = &types.TicketAction_Tclose{tclose}
	ta.Ty = types.TicketActionClose
	return policy.getWalletOperate().SendTransaction(ta, []byte("ticket"), priv, "")
}

func (policy *ticketPolicy) getTicketsByStatus(status int32) ([]*types.Ticket, [][]byte, error) {
	operater := policy.getWalletOperate()
	accounts, err := operater.GetWalletAccounts()
	if err != nil {
		return nil, nil, err
	}
	operater.GetMutex().Lock()
	defer operater.GetMutex().Unlock()
	ok, err := operater.CheckWalletStatus()
	if !ok && err != types.ErrOnlyTicketUnLocked {
		return nil, nil, err
	}
	//循环遍历所有的账户-->保证钱包已经解锁
	var tickets []*types.Ticket
	var privs [][]byte
	for _, acc := range accounts {
		t, err := policy.getTickets(acc.Addr, status)
		if err == types.ErrNotFound {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if t != nil {
			priv, err := operater.GetPrivKeyByAddr(acc.Addr)
			if err != nil {
				return nil, nil, err
			}
			privs = append(privs, priv.Bytes())
			tickets = append(tickets, t...)
		}
	}
	if len(tickets) == 0 {
		return nil, nil, types.ErrNoTicket
	}
	return tickets, privs, nil
}

func (policy *ticketPolicy) onWalletGetTickets(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletTickets)

	tickets, privs, err := policy.getTicketsByStatus(1)
	tks := &types.ReplyWalletTickets{tickets, privs}
	return topic, retty, tks, err
}

func (policy *ticketPolicy) onWalletAutoMiner(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventWalletAutoMiner)
	req, ok := msg.Data.(*types.MinerFlag)
	if !ok {
		bizlog.Error("onWalletAutoMiner", "Invalid data type.", ok)
		return topic, retty, nil, types.ErrInvalidParam
	}
	policy.store.SetAutoMinerFlag(req.Flag)
	policy.setAutoMining(req.Flag)
	policy.flushTicket()
	return topic, retty, &types.Reply{IsOk: true}, nil
}

func (policy *ticketPolicy) setAutoMining(flag int32) {
	atomic.StoreInt32(&policy.autoMinerFlag, flag)
}

func (policy *ticketPolicy) isAutoMining() bool {
	return atomic.LoadInt32(&policy.autoMinerFlag) == 1
}

func (policy *ticketPolicy) closeTicketsByAddr(height int64, priv crypto.PrivKey) ([]byte, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	tlist, err := policy.getTickets(addr, 2)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	var ids []string
	var tl []*types.Ticket
	now := types.Now().Unix()
	for _, t := range tlist {
		if !t.IsGenesis {
			if now-t.GetCreateTime() < types.GetP(height).TicketWithdrawTime {
				continue
			}
			if now-t.GetMinerTime() < types.GetP(height).TicketMinerWaitTime {
				continue
			}
		}
		tl = append(tl, t)
	}
	for i := 0; i < len(tl); i++ {
		ids = append(ids, tl[i].TicketId)
	}
	if len(ids) > 0 {
		return policy.closeTickets(priv, ids)
	}
	return nil, nil
}

func (policy *ticketPolicy) closeAllTickets(height int64) (int, error) {
	operater := policy.getWalletOperate()
	keys, err := operater.GetAllPrivKeys()
	if err != nil {
		return 0, err
	}
	var hashes [][]byte
	for _, key := range keys {
		hash, err := policy.closeTicketsByAddr(height, key)
		if err != nil {
			bizlog.Error("close Tickets By Addr", "err", err)
			continue
		}
		if hash == nil {
			continue
		}
		hashes = append(hashes, hash)
	}
	if len(hashes) > 0 {
		operater.WaitTxs(hashes)
		return len(hashes), nil
	}
	return 0, nil
}

func (policy *ticketPolicy) closeTicket(height int64) (int, error) {
	return policy.closeAllTickets(height)
}

func (policy *ticketPolicy) processFee(priv crypto.PrivKey) error {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	operater := policy.getWalletOperate()
	acc1, err := operater.GetBalance(addr, "coins")
	if err != nil {
		return err
	}
	acc2, err := operater.GetBalance(addr, "ticket")
	if err != nil {
		return err
	}
	toaddr := address.ExecAddress("ticket")
	//如果acc2 的余额足够，那题withdraw 部分钱做手续费
	if (acc1.Balance < (types.Coin / 2)) && (acc2.Balance > types.Coin) {
		_, err := operater.SendToAddress(priv, toaddr, -types.Coin, "ticket->coins", false, "")
		if err != nil {
			return err
		}
	}
	return nil
}

//手续费处理
func (policy *ticketPolicy) processFees() error {
	keys, err := policy.getWalletOperate().GetAllPrivKeys()
	if err != nil {
		return err
	}
	for _, key := range keys {
		e := policy.processFee(key)
		if e != nil {
			err = e
		}
	}
	return err
}

func (policy *ticketPolicy) withdrawFromTicketOne(priv crypto.PrivKey) ([]byte, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	operater := policy.getWalletOperate()
	acc, err := operater.GetBalance(addr, "ticket")
	if err != nil {
		return nil, err
	}
	if acc.Balance > 0 {
		hash, err := operater.SendToAddress(priv, address.ExecAddress("ticket"), -acc.Balance, "autominer->withdraw", false, "")
		if err != nil {
			return nil, err
		}
		return hash.GetHash(), nil
	}
	return nil, nil
}

func (policy *ticketPolicy) openticket(mineraddr, returnaddr string, priv crypto.PrivKey, count int32) ([]byte, error) {
	bizlog.Info("openticket", "mineraddr", mineraddr, "returnaddr", returnaddr, "count", count)
	ta := &types.TicketAction{}
	topen := &types.TicketOpen{MinerAddress: mineraddr, ReturnAddress: returnaddr, Count: count}
	ta.Value = &types.TicketAction_Topen{topen}
	ta.Ty = types.TicketActionOpen
	return policy.walletOperate.SendTransaction(ta, []byte("ticket"), priv, "")
}

func (policy *ticketPolicy) buyTicketOne(height int64, priv crypto.PrivKey) ([]byte, int, error) {
	//ticket balance and coins balance
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	operater := policy.getWalletOperate()
	acc1, err := operater.GetBalance(addr, "coins")
	if err != nil {
		return nil, 0, err
	}
	acc2, err := operater.GetBalance(addr, "ticket")
	if err != nil {
		return nil, 0, err
	}
	//留一个币作为手续费，如果手续费不够了，不能挖矿
	//判断手续费是否足够，如果不足要及时补充。
	fee := types.Coin
	if acc1.Balance+acc2.Balance-2*fee >= types.GetP(height).TicketPrice {
		// 如果可用余额+冻结余额，可以凑成新票，则转币到冻结余额
		if (acc1.Balance+acc2.Balance-2*fee)/types.GetP(height).TicketPrice > acc2.Balance/types.GetP(height).TicketPrice {
			//第一步。转移币到 ticket
			toaddr := address.ExecAddress("ticket")
			amount := acc1.Balance - 2*fee
			//必须大于0，才需要转移币
			var hash *types.ReplyHash
			if amount > 0 {
				bizlog.Info("buyTicketOne.send", "toaddr", toaddr, "amount", amount)
				hash, err = policy.walletOperate.SendToAddress(priv, toaddr, amount, "coins->ticket", false, "")

				if err != nil {
					return nil, 0, err
				}
				operater.WaitTx(hash.Hash)
			}
		}

		acc, err := operater.GetBalance(addr, "ticket")
		if err != nil {
			return nil, 0, err
		}
		count := acc.Balance / types.GetP(height).TicketPrice
		if count > 0 {
			txhash, err := policy.openticket(addr, addr, priv, int32(count))
			return txhash, int(count), err
		}
	}
	return nil, 0, nil
}

func (policy *ticketPolicy) buyTicket(height int64) ([][]byte, int, error) {
	privs, err := policy.getWalletOperate().GetAllPrivKeys()
	if err != nil {
		bizlog.Error("buyTicket.getAllPrivKeys", "err", err)
		return nil, 0, err
	}
	count := 0
	var hashes [][]byte
	for _, priv := range privs {
		hash, n, err := policy.buyTicketOne(height, priv)
		if err != nil {
			bizlog.Error("buyTicketOne", "err", err)
			continue
		}
		count += n
		if hash != nil {
			hashes = append(hashes, hash)
		}
	}
	return hashes, count, nil
}

func (policy *ticketPolicy) getMinerColdAddr(addr string) ([]string, error) {
	reqaddr := &types.ReqString{addr}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "MinerSourceList"
	req.Payload = types.Encode(reqaddr)
	api := policy.walletOperate.GetAPI()
	msg, err := api.Query(&req)
	if err != nil {
		bizlog.Error("getMinerColdAddr", "Query error", err)
		return nil, err
	}
	reply := (*msg).(*types.ReplyStrings)
	return reply.Datas, nil
}

func initMinerWhiteList(cfg *types.Wallet) {
	if len(cfg.GetMinerwhitelist()) == 0 {
		minerAddrWhiteList["*"] = true
		return
	}
	if len(cfg.GetMinerwhitelist()) == 1 && cfg.GetMinerwhitelist()[0] == "*" {
		minerAddrWhiteList["*"] = true
		return
	}
	for _, addr := range cfg.GetMinerwhitelist() {
		minerAddrWhiteList[addr] = true
	}
}

func checkMinerWhiteList(addr string) bool {
	if _, ok := minerAddrWhiteList["*"]; ok {
		return true
	}

	if _, ok := minerAddrWhiteList[addr]; ok {
		return true
	}
	return false
}

func (policy *ticketPolicy) buyMinerAddrTicketOne(height int64, priv crypto.PrivKey) ([][]byte, int, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	//判断是否绑定了coldaddr
	addrs, err := policy.getMinerColdAddr(addr)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	var hashes [][]byte
	for i := 0; i < len(addrs); i++ {
		bizlog.Info("sourceaddr", "addr", addrs[i])
		ok := checkMinerWhiteList(addrs[i])
		if !ok {
			bizlog.Info("buyMinerAddrTicketOne Cold Addr not in MinerWhiteList", "addr", addrs[i])
			continue
		}
		acc, err := policy.getWalletOperate().GetBalance(addrs[i], "ticket")
		if err != nil {
			return nil, 0, err
		}
		count := acc.Balance / types.GetP(height).TicketPrice
		if count > 0 {
			txhash, err := policy.openticket(addr, addrs[i], priv, int32(count))
			if err != nil {
				return nil, 0, err
			}
			total += int(count)
			if txhash != nil {
				hashes = append(hashes, txhash)
			}
		}
	}
	return hashes, total, nil
}

func (policy *ticketPolicy) buyMinerAddrTicket(height int64) ([][]byte, int, error) {
	privs, err := policy.getWalletOperate().GetAllPrivKeys()
	if err != nil {
		bizlog.Error("buyMinerAddrTicket.getAllPrivKeys", "err", err)
		return nil, 0, err
	}
	count := 0
	var hashes [][]byte
	for _, priv := range privs {
		hashlist, n, err := policy.buyMinerAddrTicketOne(height, priv)
		if err != nil {
			if err != types.ErrNotFound {
				bizlog.Error("buyMinerAddrTicketOne", "err", err)
			}
			continue
		}
		count += n
		if hashlist != nil {
			hashes = append(hashes, hashlist...)
		}
	}
	return hashes, count, nil
}

func (policy *ticketPolicy) withdrawFromTicket() (hashes [][]byte, err error) {
	privs, err := policy.getWalletOperate().GetAllPrivKeys()
	if err != nil {
		bizlog.Error("withdrawFromTicket.getAllPrivKeys", "err", err)
		return nil, err
	}
	for _, priv := range privs {
		hash, err := policy.withdrawFromTicketOne(priv)
		if err != nil {
			bizlog.Error("withdrawFromTicketOne", "err", err)
			continue
		}
		if hash != nil {
			hashes = append(hashes, hash)
		}
	}
	return hashes, nil
}

//检查周期 --> 10分
//开启挖矿：
//1. 自动把成熟的ticket关闭
//2. 查找超过1万余额的账户，自动购买ticket
//3. 查找mineraddress 和他对应的 账户的余额（不在1中），余额超过1万的自动购买ticket 挖矿
//
//停止挖矿：
//1. 自动把成熟的ticket关闭
//2. 查找ticket 可取的余额
//3. 取出ticket 里面的钱
func (policy *ticketPolicy) autoMining() {
	operater := policy.getWalletOperate()
	defer operater.GetWaitGroup().Done()
	cfg := operater.GetConfig()
	lastHeight := int64(0)
	for {
		select {
		case <-policy.miningTicket.C:
			if cfg.GetMinerdisable() {
				break
			}
			if !(operater.IsCaughtUp() || cfg.GetForceMining()) {
				bizlog.Error("wallet IsCaughtUp false")
				break
			}
			//判断高度是否增长
			height := operater.GetBlockHeight()
			if height <= lastHeight {
				bizlog.Error("wallet Height not inc", "height", height, "lastHeight", lastHeight)
				break
			}
			lastHeight = height
			bizlog.Info("BEG miningTicket")
			if policy.isAutoMining() {
				n1, err := policy.closeTicket(lastHeight + 1)
				if err != nil {
					bizlog.Error("closeTicket", "err", err)
				}
				err = policy.processFees()
				if err != nil {
					bizlog.Error("processFees", "err", err)
				}
				hashes1, n2, err := policy.buyTicket(lastHeight + 1)
				if err != nil {
					bizlog.Error("buyTicket", "err", err)
				}
				hashes2, n3, err := policy.buyMinerAddrTicket(lastHeight + 1)
				if err != nil {
					bizlog.Error("buyMinerAddrTicket", "err", err)
				}
				hashes := append(hashes1, hashes2...)
				if len(hashes) > 0 {
					operater.WaitTxs(hashes)
				}
				if n1+n2+n3 > 0 {
					policy.flushTicket()
				}
			} else {
				n1, err := policy.closeTicket(lastHeight + 1)
				if err != nil {
					bizlog.Error("closeTicket", "err", err)
				}
				err = policy.processFees()
				if err != nil {
					bizlog.Error("processFees", "err", err)
				}
				hashes, err := policy.withdrawFromTicket()
				if err != nil {
					bizlog.Error("withdrawFromTicket", "err", err)
				}
				if len(hashes) > 0 {
					operater.WaitTxs(hashes)
				}
				if n1 > 0 {
					policy.flushTicket()
				}
			}
			bizlog.Info("END miningTicket")
		case <-operater.GetWalletDone():
			return
		}
	}
}
