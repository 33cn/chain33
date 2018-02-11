package wallet

import (
	"errors"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/common/crypto"
	"code.aliyun.com/chain33/chain33/types"
)

func (wallet *Wallet) openticket(mineraddr, returnaddr string, priv crypto.PrivKey) error {
	ta := &types.TicketAction{}
	topen := &types.TicketOpen{MinerAddress: mineraddr, ReturnAddress: returnaddr, Count: 1}
	ta.Value = &types.TicketAction_Topen{topen}
	ta.Ty = types.TicketActionOpen
	err := wallet.sendTransactionWait(ta, []byte("ticket"), priv, "")
	if err != nil {
		return err
	}
	return nil
}

func (wallet *Wallet) bindminer(mineraddr, returnaddr string, priv crypto.PrivKey) error {
	ta := &types.TicketAction{}
	tbind := &types.TicketBind{MinerAddress: mineraddr, ReturnAddress: returnaddr}
	ta.Value = &types.TicketAction_Tbind{tbind}
	ta.Ty = types.TicketActionBind
	err := wallet.sendTransactionWait(ta, []byte("ticket"), priv, "")
	if err != nil {
		return err
	}
	return nil
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
	tx.Fee += types.MinFee
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
	for {
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

func (client *Wallet) queryBalance(in *types.ReqBalance) ([]*types.Account, error) {

	switch in.GetExecer() {
	case "coins":
		addrs := in.GetAddresses()
		var exaddrs []string
		for _, addr := range addrs {
			if err := account.CheckAddress(addr); err != nil {
				addr = account.ExecAddress(addr).String()
				exaddrs = append(exaddrs, addr)
			}
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
