package wallet

import (
	"encoding/hex"
	"errors"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
)

func (wallet *Wallet) openticket(mineraddr, returnaddr string, priv crypto.PrivKey, count int32) ([]byte, error) {
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
func (wallet *Wallet) closeTickets(priv crypto.PrivKey, ids []string) error {
	for i := 0; i < len(ids); i += 100 {
		end := i + 100
		if end > len(ids) {
			end = len(ids)
		}
		ta := &types.TicketAction{}
		tclose := &types.TicketClose{ids[i:end]}
		ta.Value = &types.TicketAction_Tclose{tclose}
		ta.Ty = types.TicketActionClose
		_, err := wallet.sendTransaction(ta, []byte("ticket"), priv, "")
		if err != nil {
			return err
		}
	}
	return nil
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
	if !ok {
		return nil, nil, err
	}
	//循环遍历所有的账户-->保证钱包已经解锁
	var tickets []*types.Ticket
	var privs [][]byte
	for _, account := range accounts.Wallets {
		t, err := wallet.getTickets(account.Acc.Addr, status)
		if err != nil {
			return nil, nil, err
		}
		if t != nil {
			priv, err := wallet.getPrivKeyByAddr(account.Acc.Addr)
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
	if !ok {
		return nil, err
	}
	var privs []crypto.PrivKey
	for _, account := range accounts.Wallets {
		priv, err := wallet.getPrivKeyByAddr(account.Acc.Addr)
		if err != nil {
			return nil, err
		}
		privs = append(privs, priv)
	}
	return privs, nil
}

func (wallet *Wallet) closeAllTickets() error {
	keys, err := wallet.getAllPrivKeys()
	if err != nil {
		return err
	}
	for _, key := range keys {
		err = wallet.closeTicketsByAddr(key)
		if err != nil {
			walletlog.Error("close Tickets By Addr", "err", err)
			return err
		}
	}
	return nil
}

func (wallet *Wallet) withdrawFromTicketOne(priv crypto.PrivKey) error {
	addr := account.PubKeyToAddress(priv.PubKey().Bytes()).String()
	acc, err := wallet.getBalance(addr, "ticket")
	if err != nil {
		return err
	}
	if acc.Balance > 0 {
		_, err := wallet.sendToAddress(priv, account.ExecAddress("ticket").String(), -acc.Balance, "autominer->withdraw")
		if err != nil {
			return err
		}
	}
	return nil
}

func (wallet *Wallet) buyTicketOne(priv crypto.PrivKey) error {
	//ticket balance and coins balance
	addr := account.PubKeyToAddress(priv.PubKey().Bytes()).String()
	acc1, err := wallet.getBalance(addr, "coins")
	if err != nil {
		return err
	}
	acc2, err := wallet.getBalance(addr, "ticket")
	if err != nil {
		return err
	}
	//留一个币作为手续费，如果手续费不够了，不能挖矿
	//判断手续费是否足够，如果不足要及时补充。
	fee := types.Coin
	if acc1.Balance+acc2.Balance -2*fee >= types.TicketPrice {
		//第一步。转移币到 ticket
		toaddr := account.ExecAddress("ticket").String()
		amount := acc1.Balance - 2*fee
		//必须大于0，才需要转移币
		var hash *types.ReplyHash
		if amount > 0 {
			hash, err = wallet.sendToAddress(priv, toaddr, amount, "coins->ticket")
			if err != nil {
				return err
			}
			wallet.waitTx(hash.Hash)
		}
		acc, err := wallet.getBalance(addr, "ticket")
		if err != nil {
			return err
		}
		count := acc.Balance / types.TicketPrice
		if count > 0 {
			_, err := wallet.openticket(addr, addr, priv, int32(count))
			return err
		}
	}
	return nil
}

func (wallet *Wallet) buyMinerAddrTicketOne(priv crypto.PrivKey) error {
	addr := account.PubKeyToAddress(priv.PubKey().Bytes()).String()
	//判断是否绑定了coldaddr
	addrs, err := wallet.getMinerSourceList(addr)
	if err != nil {
		return err
	}
	for i := 0; i < len(addrs); i++ {
		walletlog.Error("sourceaddr", "addr", addrs[i])
		acc, err := wallet.getBalance(addrs[i], "ticket")
		if err != nil {
			return err
		}
		if acc.Balance >= types.TicketPrice {
			count := acc.Balance / types.TicketPrice
			if count > 0 {
				_, err := wallet.openticket(addr, addrs[i], priv, int32(count))
				return err
			}
		}
	}
	return nil
}

func (wallet *Wallet) processFee(priv crypto.PrivKey) error {
	addr := account.PubKeyToAddress(priv.PubKey().Bytes()).String()
	acc1, err := wallet.getBalance(addr, "coins")
	if err != nil {
		return err
	}
	acc2, err := wallet.getBalance(addr, "ticket")
	if err != nil {
		return err
	}
	toaddr := account.ExecAddress("ticket").String()
	//如果acc2 的余额足够，那题withdraw 部分钱做手续费
	if (acc1.Balance < (types.Coin / 2)) && (acc2.Balance > types.Coin) {
		_, err := wallet.sendToAddress(priv, toaddr, -types.Coin, "ticket->coins")
		if err != nil {
			return err
		}
	}
	return nil
}

func (wallet *Wallet) closeTicketsByAddr(priv crypto.PrivKey) error {
	wallet.processFee(priv)
	addr := account.PubKeyToAddress(priv.PubKey().Bytes()).String()
	tlist, err := wallet.getTickets(addr, 2)
	if err != nil && err != types.ErrNotFound {
		return err
	}
	if len(tlist) == 0 {
		return nil
	}
	now := time.Now().Unix()
	var ids []string
	for i := 0; i < len(tlist); i++ {
		if now-tlist[i].CreateTime > types.TicketWithdrawTime {
			ids = append(ids, tlist[i].TicketId)
		}
	}
	if len(ids) > 1 {
		wallet.closeTickets(priv, ids)
	}
	return nil
}

func (client *Wallet) getTickets(addr string, status int32) ([]*types.Ticket, error) {
	reqaddr := &types.TicketList{addr, status}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "TicketList"
	req.Payload = types.Encode(reqaddr)
	msg := client.qclient.NewMessage("blockchain", types.EventQuery, &req)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
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
		to = account.ExecAddress(string(execer)).String()
	}
	tx := &types.Transaction{Execer: execer, Payload: types.Encode(payload), Fee: 1e6, To: to}
	tx.Nonce = wallet.random.Int63()
	tx.Fee, err = tx.GetRealFee()
	if err != nil {
		return nil, err
	}
	tx.Sign(types.SECP256K1, priv)
	reply, err := wallet.sendTx(tx)
	if err != nil {
		return nil, err
	}
	if !reply.IsOk {
		walletlog.Info("wallet sendTransaction", "err", reply.GetMsg())
		return nil, errors.New(string(reply.GetMsg()))
	}
	return tx.Hash(), nil
}

func (wallet *Wallet) sendTx(tx *types.Transaction) (*types.Reply, error) {
	if wallet.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := wallet.qclient.NewMessage("mempool", types.EventTx, tx)
	err := wallet.qclient.Send(msg, true)
	if err != nil {
		walletlog.Error("SendTx", "Error", err.Error())
		return nil, err
	}
	resp, err := wallet.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.Reply), nil
}

func (wallet *Wallet) waitTx(hash []byte) *types.TransactionDetail {
	i := 0
	for {
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

func (client *Wallet) queryTx(hash []byte) (*types.TransactionDetail, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventQueryTx, &types.ReqHash{hash})
	err := client.qclient.Send(msg, true)
	if err != nil {
		walletlog.Error("QueryTx", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.TransactionDetail), nil
}

func (wallet *Wallet) sendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string) (*types.ReplyHash, error) {
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
	//初始化随机数d
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: wallet.FeeAmount, To: addrto, Nonce: wallet.random.Int63()}
	tx.Sign(types.SECP256K1, priv)

	//发送交易信息给mempool模块
	msg := wallet.qclient.NewMessage("mempool", types.EventTx, tx)
	wallet.qclient.Send(msg, true)
	resp, err := wallet.qclient.Wait(msg)
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

func (client *Wallet) queryBalance(in *types.ReqBalance) ([]*types.Account, error) {

	switch in.GetExecer() {
	case "coins":
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := account.CheckAddress(addr); err != nil {
				addr = account.ExecAddress(addr).String()
			}
			exaddrs = append(exaddrs, addr)
		}
		accounts, err := account.LoadAccounts(client.q, exaddrs)
		if err != nil {
			walletlog.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := account.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {
			account, err := account.LoadExecAccountQueue(client.q, addr, execaddress.String())
			if err != nil {
				walletlog.Error("GetBalance", "err", err.Error())
				return nil, err
			}

			accounts = append(accounts, account)
		}
		return accounts, nil
	}
	return nil, nil
}

func (client *Wallet) getMinerSourceList(addr string) ([]string, error) {
	reqaddr := &types.ReqString{addr}
	var req types.Query
	req.Execer = []byte("ticket")
	req.FuncName = "MinerSourceList"
	req.Payload = types.Encode(reqaddr)

	msg := client.qclient.NewMessage("blockchain", types.EventQuery, &req)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	reply := resp.GetData().(types.Message).(*types.ReplyStrings)
	return reply.Datas, nil
}
