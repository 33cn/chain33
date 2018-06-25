package wallet

import (
	"encoding/hex"
	"errors"
	"sync/atomic"
	"time"

	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/types"
)

func (wallet *Wallet) openticket(mineraddr, returnaddr string, priv crypto.PrivKey, count int32) ([]byte, error) {
	walletlog.Info("openticket", "mineraddr", mineraddr, "returnaddr", returnaddr, "count", count)
	ta := &types.TicketAction{}
	topen := &types.TicketOpen{MinerAddress: mineraddr, ReturnAddress: returnaddr, Count: count}
	ta.Value = &types.TicketAction_Topen{topen}
	ta.Ty = types.TicketActionOpen
	return wallet.sendTransaction(ta, []byte("ticket"), priv, "")
}

func (wallet *Wallet) bindminer(mineraddr, returnaddr string, priv crypto.PrivKey) ([]byte, error) {
	ta := &types.TicketAction{}
	tbind := &types.TicketBind{MinerAddress: mineraddr, ReturnAddress: returnaddr}
	ta.Value = &types.TicketAction_Tbind{tbind}
	ta.Ty = types.TicketActionBind
	return wallet.sendTransaction(ta, []byte("ticket"), priv, "")
}

//通过rpc 精选close 操作
func (wallet *Wallet) closeTickets(priv crypto.PrivKey, ids []string) ([]byte, error) {
	//每次最多close 200个
	end := 200
	if end > len(ids) {
		end = len(ids)
	}
	walletlog.Info("closeTickets", "ids", ids[0:end])
	ta := &types.TicketAction{}
	tclose := &types.TicketClose{ids[0:end]}
	ta.Value = &types.TicketAction_Tclose{tclose}
	ta.Ty = types.TicketActionClose
	return wallet.sendTransaction(ta, []byte("ticket"), priv, "")
}

func (wallet *Wallet) getBalance(addr string, execer string) (*types.Account, error) {
	reqbalance := &types.ReqBalance{Addresses: []string{addr}, Execer: execer}
	reply, err := wallet.queryBalance(reqbalance)
	if err != nil {
		return nil, err
	}
	return reply[0], nil
}

func (wallet *Wallet) GetTickets(status int32) ([]*types.Ticket, [][]byte, error) {
	accounts, err := wallet.ProcGetAccountList()
	if err != nil {
		return nil, nil, err
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	ok, err := wallet.CheckWalletStatus()
	if !ok && err != types.ErrOnlyTicketUnLocked {
		return nil, nil, err
	}
	//循环遍历所有的账户-->保证钱包已经解锁
	var tickets []*types.Ticket
	var privs [][]byte
	for _, acc := range accounts.Wallets {
		t, err := wallet.getTickets(acc.Acc.Addr, status)
		if err == types.ErrNotFound {
			continue
		}
		if err != nil {
			return nil, nil, err
		}
		if t != nil {
			priv, err := wallet.getPrivKeyByAddr(acc.Acc.Addr)
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

func (wallet *Wallet) getAllPrivKeys() ([]crypto.PrivKey, error) {
	accounts, err := wallet.ProcGetAccountList()
	if err != nil {
		return nil, err
	}
	wallet.mtx.Lock()
	defer wallet.mtx.Unlock()
	ok, err := wallet.CheckWalletStatus()
	if !ok && err != types.ErrOnlyTicketUnLocked {
		return nil, err
	}
	var privs []crypto.PrivKey
	for _, acc := range accounts.Wallets {
		priv, err := wallet.getPrivKeyByAddr(acc.Acc.Addr)
		if err != nil {
			return nil, err
		}
		privs = append(privs, priv)
	}
	return privs, nil
}

func (wallet *Wallet) GetHeight() int64 {
	msg := wallet.client.NewMessage("blockchain", types.EventGetBlockHeight, nil)
	wallet.client.Send(msg, true)
	replyHeight, err := wallet.client.Wait(msg)
	h := replyHeight.GetData().(*types.ReplyBlockHeight).Height
	walletlog.Debug("getheight = ", "height", h)
	if err != nil {
		return 0
	}
	return h
}

//手续费处理
func (wallet *Wallet) processFees() error {
	keys, err := wallet.getAllPrivKeys()
	if err != nil {
		return err
	}
	for _, key := range keys {
		e := wallet.processFee(key)
		if e != nil {
			err = e
		}
	}
	return err
}

func (wallet *Wallet) closeAllTickets(height int64) (int, error) {
	keys, err := wallet.getAllPrivKeys()
	if err != nil {
		return 0, err
	}
	var hashes [][]byte
	for _, key := range keys {
		hash, err := wallet.closeTicketsByAddr(height, key)
		if err != nil {
			walletlog.Error("close Tickets By Addr", "err", err)
			continue
		}
		if hash == nil {
			continue
		}
		hashes = append(hashes, hash)
	}
	if len(hashes) > 0 {
		wallet.waitTxs(hashes)
		return len(hashes), nil
	}
	return 0, nil
}

func (wallet *Wallet) forceCloseAllTicket(height int64) (*types.ReplyHashes, error) {
	keys, err := wallet.getAllPrivKeys()
	if err != nil {
		return nil, err
	}
	var hashes types.ReplyHashes
	for _, key := range keys {
		hash, err := wallet.forceCloseTicketsByAddr(height, key)
		if err != nil {
			walletlog.Error("close Tickets By Addr", "err", err)
			continue
		}
		if hash == nil {
			continue
		}
		hashes.Hashes = append(hashes.Hashes, hash)
	}
	return &hashes, nil
}

func (wallet *Wallet) withdrawFromTicketOne(priv crypto.PrivKey) ([]byte, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	acc, err := wallet.getBalance(addr, "ticket")
	if err != nil {
		return nil, err
	}
	if acc.Balance > 0 {
		hash, err := wallet.sendToAddress(priv, address.ExecAddress("ticket"), -acc.Balance, "autominer->withdraw", false, "")
		if err != nil {
			return nil, err
		}
		return hash.GetHash(), nil
	}
	return nil, nil
}

func (wallet *Wallet) buyTicketOne(height int64, priv crypto.PrivKey) ([]byte, int, error) {
	//ticket balance and coins balance
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	acc1, err := wallet.getBalance(addr, "coins")
	if err != nil {
		return nil, 0, err
	}
	acc2, err := wallet.getBalance(addr, "ticket")
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
				walletlog.Info("buyTicketOne.send", "toaddr", toaddr, "amount", amount)
				hash, err = wallet.sendToAddress(priv, toaddr, amount, "coins->ticket", false, "")

				if err != nil {
					return nil, 0, err
				}
				wallet.waitTx(hash.Hash)
			}
		}

		acc, err := wallet.getBalance(addr, "ticket")
		if err != nil {
			return nil, 0, err
		}
		count := acc.Balance / types.GetP(height).TicketPrice
		if count > 0 {
			txhash, err := wallet.openticket(addr, addr, priv, int32(count))
			return txhash, int(count), err
		}
	}
	return nil, 0, nil
}

func (wallet *Wallet) buyMinerAddrTicketOne(height int64, priv crypto.PrivKey) ([][]byte, int, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	//判断是否绑定了coldaddr
	addrs, err := wallet.getMinerColdAddr(addr)
	if err != nil {
		return nil, 0, err
	}
	total := 0
	var hashes [][]byte
	for i := 0; i < len(addrs); i++ {
		walletlog.Info("sourceaddr", "addr", addrs[i])
		acc, err := wallet.getBalance(addrs[i], "ticket")
		if err != nil {
			return nil, 0, err
		}
		count := acc.Balance / types.GetP(height).TicketPrice
		if count > 0 {
			txhash, err := wallet.openticket(addr, addrs[i], priv, int32(count))
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

func (wallet *Wallet) processFee(priv crypto.PrivKey) error {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	acc1, err := wallet.getBalance(addr, "coins")
	if err != nil {
		return err
	}
	acc2, err := wallet.getBalance(addr, "ticket")
	if err != nil {
		return err
	}
	toaddr := address.ExecAddress("ticket")
	//如果acc2 的余额足够，那题withdraw 部分钱做手续费
	if (acc1.Balance < (types.Coin / 2)) && (acc2.Balance > types.Coin) {
		_, err := wallet.sendToAddress(priv, toaddr, -types.Coin, "ticket->coins", false, "")
		if err != nil {
			return err
		}
	}
	return nil
}

func (wallet *Wallet) closeTicketsByAddr(height int64, priv crypto.PrivKey) ([]byte, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	tlist, err := wallet.getTickets(addr, 2)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	var ids []string
	var tl []*types.Ticket
	now := time.Now().Unix()
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
		return wallet.closeTickets(priv, ids)
	}
	return nil, nil
}

func (wallet *Wallet) forceCloseTicketsByAddr(height int64, priv crypto.PrivKey) ([]byte, error) {
	addr := address.PubKeyToAddress(priv.PubKey().Bytes()).String()
	tlist1, err1 := wallet.getTickets(addr, 1)
	if err1 != nil && err1 != types.ErrNotFound {
		return nil, err1
	}
	tlist2, err2 := wallet.getTickets(addr, 2)
	if err2 != nil && err2 != types.ErrNotFound {
		return nil, err1
	}
	tlist := append(tlist1, tlist2...)
	var ids []string
	var tl []*types.Ticket
	now := time.Now().Unix()
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
		return wallet.closeTickets(priv, ids)
	}
	return nil, nil
}

func (wallet *Wallet) getTickets(addr string, status int32) ([]*types.Ticket, error) {
	reqaddr := &types.TicketList{addr, status}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "TicketList"
	req.Payload = types.Encode(reqaddr)
	msg := wallet.client.NewMessage("blockchain", types.EventQuery, &req)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	reply := resp.GetData().(types.Message).(*types.ReplyTicketList)
	return reply.Tickets, nil
}

func (wallet *Wallet) sendTransactionWait(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (err error) {
	hash, err := wallet.sendTransaction(payload, execer, priv, to)
	if err != nil {
		return err
	}
	txinfo := wallet.waitTx(hash)
	if txinfo.Receipt.Ty != types.ExecOk {
		return errors.New("sendTransactionWait error")
	}
	return nil
}

func (wallet *Wallet) sendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	if to == "" {
		to = address.ExecAddress(string(execer))
	}
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(payload), Fee: minFee, To: to}
	tx.Nonce = wallet.random.Int63()
	tx.Fee, err = tx.GetRealFee(wallet.getFee())
	if err != nil {
		return nil, err
	}
	tx.SetExpire(time.Second * 120)
	tx.Sign(int32(SignType), priv)
	reply, err := wallet.sendTx(tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		walletlog.Info("wallet sendTransaction", "err", string(reply.GetMsg()))
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func (wallet *Wallet) sendTx(tx *types.Transaction) (*types.Reply, error) {
	if wallet.client == nil {
		panic("client not bind message queue.")
	}
	msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
	err := wallet.client.Send(msg, true)
	if err != nil {
		walletlog.Error("SendTx", "Error", err.Error())
		return nil, err
	}
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.Reply), nil
}

func (wallet *Wallet) waitTx(hash []byte) *types.TransactionDetail {
	i := 0
	for {
		if atomic.LoadInt32(&wallet.isclosed) == 1 {
			return nil
		}
		i++
		if i%100 == 0 {
			walletlog.Error("wait transaction timeout", "hash", hex.EncodeToString(hash))
			return nil
		}
		res, err := wallet.queryTx(hash)
		if err != nil {
			time.Sleep(time.Second)
		}
		if res != nil {
			return res
		}
	}
}

func (wallet *Wallet) waitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	for _, hash := range hashes {
		result := wallet.waitTx(hash)
		ret = append(ret, result)
	}
	return ret
}

func (wallet *Wallet) queryTx(hash []byte) (*types.TransactionDetail, error) {
	msg := wallet.client.NewMessage("blockchain", types.EventQueryTx, &types.ReqHash{hash})
	err := wallet.client.Send(msg, true)
	if err != nil {
		walletlog.Error("QueryTx", "Error", err.Error())
		return nil, err
	}
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.TransactionDetail), nil
}

func (wallet *Wallet) sendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	var tx *types.Transaction
	if !Istoken {
		transfer := &types.CoinsAction{}
		if amount > 0 {
			v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: amount, Note: note}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionTransfer
		} else {
			v := &types.CoinsAction_Withdraw{&types.CoinsWithdraw{Amount: -amount, Note: note}}
			transfer.Value = v
			transfer.Ty = types.CoinsActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), To: addrto, Nonce: wallet.random.Int63()}
	} else {
		transfer := &types.TokenAction{}
		if amount > 0 {
			v := &types.TokenAction_Transfer{&types.CoinsTransfer{Cointoken: tokenSymbol, Amount: amount, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Withdraw{&types.CoinsWithdraw{Cointoken: tokenSymbol, Amount: -amount, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("token"), Payload: types.Encode(transfer), To: addrto, Nonce: wallet.random.Int63()}
	}
	tx.SetExpire(time.Second * 120)
	fee, err := tx.GetRealFee(wallet.getFee())
	if err != nil {
		return nil, err
	}
	tx.Fee = fee
	tx.Sign(int32(SignType), priv)

	//发送交易信息给mempool模块
	msg := wallet.client.NewMessage("mempool", types.EventTx, tx)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("ProcSendToAddress", "Send err", err)
		return nil, err
	}
	reply := resp.GetData().(*types.Reply)
	if !reply.GetIsOk() {
		return nil, errors.New(string(reply.GetMsg()))
	}
	var hash types.ReplyHash
	hash.Hash = tx.Hash()
	return &hash, nil
}

func (wallet *Wallet) queryBalance(in *types.ReqBalance) ([]*types.Account, error) {

	switch in.GetExecer() {
	case "coins":
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := address.CheckAddress(addr); err != nil {
				addr = address.ExecAddress(addr)
			}
			exaddrs = append(exaddrs, addr)
		}
		accounts, err := accountdb.LoadAccounts(wallet.api, exaddrs)
		if err != nil {
			walletlog.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := address.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			acc, err := accountdb.LoadExecAccountQueue(wallet.api, addr, execaddress)
			if err != nil {
				walletlog.Error("GetBalance", "err", err.Error())
				return nil, err
			}
			accounts = append(accounts, acc)
		}
		return accounts, nil
	}
}

func (wallet *Wallet) getMinerColdAddr(addr string) ([]string, error) {
	reqaddr := &types.ReqString{addr}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "MinerSourceList"
	req.Payload = types.Encode(reqaddr)

	msg := wallet.client.NewMessage("blockchain", types.EventQuery, &req)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return nil, err
	}
	reply := resp.GetData().(types.Message).(*types.ReplyStrings)
	return reply.Datas, nil
}

func (wallet *Wallet) IsCaughtUp() bool {
	if wallet.client == nil {
		panic("wallet client not bind message queue.")
	}
	msg := wallet.client.NewMessage("blockchain", types.EventIsSync, nil)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		return false
	}
	return resp.GetData().(*types.IsCaughtUp).GetIscaughtup()
}
