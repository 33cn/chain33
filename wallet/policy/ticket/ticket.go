package ticket

import (
	"encoding/hex"
	"time"

	"github.com/inconshreveable/log15"

	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
	wcom "gitlab.33.cn/chain33/chain33/wallet/common"
)

var (
	bizlog = log15.New("module", "wallet.ticket")
)

func init() {
	wcom.RegisterPolicy(types.TicketX, New())
}

func New() wcom.WalletBizPolicy {
	return &ticketPolicy{}
}

type ticketPolicy struct {
	walletOperate wcom.WalletOperate
	store         *ticketStore
	needFlush     bool
}

func (policy *ticketPolicy) initFuncMap(walletOperate wcom.WalletOperate) {
	wcom.RegisterMsgFunc(types.EventCloseTickets, policy.onCloseTickets)
	wcom.RegisterMsgFunc(types.EventWalletGetTickets, policy.onWalletGetTickets)
}

func (policy *ticketPolicy) Init(walletBiz wcom.WalletOperate) {
	policy.walletOperate = walletBiz
	policy.store = NewStore(walletBiz.GetDBStore())
	policy.needFlush = false
	policy.initFuncMap(walletBiz)
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

func (policy *ticketPolicy) OnCreateNewAccount(acc *types.Account) {
}

func (policy *ticketPolicy) OnImportPrivateKey(acc *types.Account) {
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
	api := policy.walletOperate.GetAPI()
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

func (policy *ticketPolicy) queryTx(hash []byte) (*types.TransactionDetail, error) {
	api := policy.walletOperate.GetAPI()
	return api.QueryTx(&types.ReqHash{hash})
}

func (policy *ticketPolicy) waitTx(hash []byte) *types.TransactionDetail {
	i := 0
	for {
		if policy.walletOperate.IsClose() {
			return nil
		}
		i++
		if i%100 == 0 {
			bizlog.Error("wait transaction timeout", "hash", hex.EncodeToString(hash))
			return nil
		}
		res, err := policy.queryTx(hash)
		if err != nil {
			time.Sleep(time.Second)
		}
		if res != nil {
			return res
		}
	}
}

func (policy *ticketPolicy) onCloseTickets(msg *queue.Message) (string, int64, interface{}, error) {
	topic := "rpc"
	retty := int64(types.EventReplyHashes)

	reply, err := policy.forceCloseTicket(policy.walletOperate.GetBlockHeight() + 1)
	if err != nil {
		bizlog.Error("onCloseTickets", "forceCloseTicket error", err.Error())
	} else {
		go func() {
			if len(reply.Hashes) > 0 {
				policy.waitTxs(reply.Hashes)
				policy.flushTicket()
			}
		}()
	}
	return topic, retty, reply, err
}

func (policy *ticketPolicy) waitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	for _, hash := range hashes {
		result := policy.waitTx(hash)
		ret = append(ret, result)
	}
	return ret
}

func (policy *ticketPolicy) forceCloseTicket(height int64) (*types.ReplyHashes, error) {
	return policy.forceCloseAllTicket(height)
}

func (policy *ticketPolicy) forceCloseAllTicket(height int64) (*types.ReplyHashes, error) {
	keys, err := policy.walletOperate.GetAllPrivKeys()
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
	api := policy.walletOperate.GetAPI()
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
	return policy.walletOperate.SendTransaction(ta, []byte("ticket"), priv, "")
}

func (policy *ticketPolicy) getTicketsByStatus(status int32) ([]*types.Ticket, [][]byte, error) {
	accounts, err := policy.walletOperate.GetWalletAccounts()
	if err != nil {
		return nil, nil, err
	}
	policy.walletOperate.GetMutex().Lock()
	defer policy.walletOperate.GetMutex().Unlock()
	ok, err := policy.walletOperate.CheckWalletStatus()
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
			priv, err := policy.walletOperate.GetPrivKeyByAddr(acc.Addr)
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
