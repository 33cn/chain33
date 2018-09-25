package wallet

import (
	"encoding/hex"
	"errors"
	"sync/atomic"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	cty "gitlab.33.cn/chain33/chain33/system/dapp/coins/types"
	"gitlab.33.cn/chain33/chain33/types"
)

func (wallet *Wallet) bindminer(mineraddr, returnaddr string, priv crypto.PrivKey) ([]byte, error) {
	ta := &types.TicketAction{}
	tbind := &types.TicketBind{MinerAddress: mineraddr, ReturnAddress: returnaddr}
	ta.Value = &types.TicketAction_Tbind{tbind}
	ta.Ty = types.TicketActionBind
	return wallet.sendTransaction(ta, []byte("ticket"), priv, "")
}

func (wallet *Wallet) GetBalance(addr string, execer string) (*types.Account, error) {
	return wallet.getBalance(addr, execer)
}

func (wallet *Wallet) getBalance(addr string, execer string) (*types.Account, error) {
	reqbalance := &types.ReqBalance{Addresses: []string{addr}, Execer: execer}
	reply, err := wallet.queryBalance(reqbalance)
	if err != nil {
		return nil, err
	}
	return reply[0], nil
}

func (wallet *Wallet) GetAllPrivKeys() ([]crypto.PrivKey, error) {
	return wallet.getAllPrivKeys()
}

func (wallet *Wallet) getAllPrivKeys() ([]crypto.PrivKey, error) {
	accounts, err := wallet.GetWalletAccounts()
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
	for _, acc := range accounts {
		priv, err := wallet.getPrivKeyByAddr(acc.Addr)
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

func (wallet *Wallet) SendTransaction(payload types.Message, execer []byte, priv crypto.PrivKey, to string) (hash []byte, err error) {
	return wallet.sendTransaction(payload, execer, priv, to)
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

func (wallet *Wallet) WaitTx(hash []byte) *types.TransactionDetail {
	return wallet.waitTx(hash)
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

func (wallet *Wallet) WaitTxs(hashes [][]byte) (ret []*types.TransactionDetail) {
	return wallet.waitTxs(hashes)
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
func (wallet *Wallet) SendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	return wallet.sendToAddress(priv, addrto, amount, note, Istoken, tokenSymbol)
}

func (wallet *Wallet) sendToAddress(priv crypto.PrivKey, addrto string, amount int64, note string, Istoken bool, tokenSymbol string) (*types.ReplyHash, error) {
	var tx *types.Transaction
	if !Istoken {
		transfer := &cty.CoinsAction{}
		if amount > 0 {
			v := &cty.CoinsAction_Transfer{&types.AssetsTransfer{Amount: amount, Note: note}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionTransfer
		} else {
			v := &cty.CoinsAction_Withdraw{&types.AssetsWithdraw{Amount: -amount, Note: note}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionWithdraw
		}
		tx = &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), To: addrto, Nonce: wallet.random.Int63()}
	} else {
		transfer := &types.TokenAction{}
		if amount > 0 {
			v := &types.TokenAction_Transfer{&types.AssetsTransfer{Cointoken: tokenSymbol, Amount: amount, Note: note}}
			transfer.Value = v
			transfer.Ty = types.ActionTransfer
		} else {
			v := &types.TokenAction_Withdraw{&types.AssetsWithdraw{Cointoken: tokenSymbol, Amount: -amount, Note: note}}
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
	reply, err := wallet.api.IsSync()
	if err != nil {
		return false
	}
	return reply.IsOk
}

func (wallet *Wallet) GetRofPrivateTx(txhashptr *string) (R_txpubkey []byte, err error) {
	txhash, err := common.FromHex(*txhashptr)
	if err != nil {
		walletlog.Error("GetRofPrivateTx common.FromHex", "err", err)
		return nil, err
	}
	var reqHashes types.ReqHashes
	reqHashes.Hashes = append(reqHashes.Hashes, txhash)

	//通过txhashs获取对应的txdetail
	msg := wallet.client.NewMessage("blockchain", types.EventGetTransactionByHash, &reqHashes)
	wallet.client.Send(msg, true)
	resp, err := wallet.client.Wait(msg)
	if err != nil {
		walletlog.Error("GetRofPrivateTx EventGetTransactionByHash", "err", err)
		return nil, err
	}
	TxDetails := resp.GetData().(*types.TransactionDetails)
	if TxDetails == nil {
		walletlog.Error("GetRofPrivateTx TransactionDetails is nil")
		return nil, errors.New("ErrTxDetail")
	}
	if len(TxDetails.Txs) <= 0 {
		walletlog.Error("GetRofPrivateTx TransactionDetails is empty")
		return nil, errors.New("ErrTxDetail")
	}

	if "privacy" != string(TxDetails.Txs[0].Tx.Execer) {
		walletlog.Error("GetRofPrivateTx get tx but not privacy")
		return nil, errors.New("ErrPrivacyExecer")
	}

	var privateAction types.PrivacyAction
	if err := types.Decode(TxDetails.Txs[0].Tx.Payload, &privateAction); err != nil {
		walletlog.Error("GetRofPrivateTx failed to decode payload")
		return nil, errors.New("ErrPrivacyPayload")
	}

	if types.ActionPublic2Privacy == privateAction.Ty {
		return privateAction.GetPublic2Privacy().GetOutput().GetRpubKeytx(), nil
	} else if types.ActionPrivacy2Privacy == privateAction.Ty {
		return privateAction.GetPrivacy2Privacy().GetOutput().GetRpubKeytx(), nil
	} else {
		walletlog.Info("GetPrivateTxByHashes failed to get value required", "privacy type is", privateAction.Ty)
		return nil, errors.New("ErrPrivacyType")
	}
}
