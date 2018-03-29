package rpc

import (
	"errors"
	"math/rand"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

//提供系统rpc接口

var accountdb = account.NewCoinsAccount()

type channelClient struct {
	queue.Client
}

func (c *channelClient) CreateRawTransaction(parm *types.CreateTx) ([]byte, error) {
	if parm == nil {
		return nil, errors.New("parm is null")
	}
	v := &types.CoinsAction_Transfer{&types.CoinsTransfer{Amount: parm.GetAmount(), Note: parm.GetNote()}}
	transfer := &types.CoinsAction{Value: v, Ty: types.CoinsActionTransfer}

	//初始化随机数
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx := &types.Transaction{Execer: []byte("coins"), Payload: types.Encode(transfer), Fee: parm.GetFee(), To: parm.GetTo(), Nonce: r.Int63()}
	data := types.Encode(tx)
	return data, nil

}

func (c *channelClient) SendRawTransaction(parm *types.SignedTx) queue.Message {
	var tx types.Transaction
	err := types.Decode(parm.GetUnsign(), &tx)

	if err == nil {
		tx.Signature = &types.Signature{parm.GetTy(), parm.GetPubkey(), parm.GetSign()}
		msg := c.NewMessage("mempool", types.EventTx, &tx)
		err := c.Send(msg, true)
		if err != nil {
			var msg queue.Message
			log.Error("SendRawTransaction", "Error", err.Error())
			msg.Data = err
			return msg
		}
		resp, err := c.Wait(msg)

		if err != nil {

			resp.Data = err

		}
		if resp.GetData().(*types.Reply).GetIsOk() {
			resp.GetData().(*types.Reply).Msg = tx.Hash()
		}

		return resp
	}
	var msg queue.Message
	msg.Data = err
	return msg

}

//channel
func (c *channelClient) SendTx(tx *types.Transaction) queue.Message {
	if c == nil {
		panic("c not bind message queue.")
	}
	msg := c.NewMessage("mempool", types.EventTx, tx)
	err := c.Send(msg, true)
	if err != nil {
		var msg queue.Message
		log.Error("SendTx", "Error", err.Error())
		msg.Data = err
		return msg
	}
	resp, err := c.Wait(msg)
	if err != nil {

		resp.Data = err
	}
	if resp.GetData().(*types.Reply).GetIsOk() {
		resp.GetData().(*types.Reply).Msg = tx.Hash()
	}
	return resp
}

func (c *channelClient) GetBlocks(start int64, end int64, isdetail bool) (*types.BlockDetails, error) {
	msg := c.NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, end, isdetail, []string{""}})
	err := c.Send(msg, true)
	if err != nil {

		log.Error("SendRawTransaction", "Error", err.Error())

		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.BlockDetails), nil
}

func (c *channelClient) QueryTx(hash []byte) (*types.TransactionDetail, error) {
	msg := c.NewMessage("blockchain", types.EventQueryTx, &types.ReqHash{hash})
	err := c.Send(msg, true)
	if err != nil {
		log.Error("QueryTx", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.TransactionDetail), nil
}

func (c *channelClient) GetLastHeader() (*types.Header, error) {
	msg := c.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetLastHeader", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Header), nil
}

func (c *channelClient) GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfos, error) {
	msg := c.NewMessage("blockchain", types.EventGetTransactionByAddr, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetTxByAddr", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyTxInfos), nil
}

func (c *channelClient) GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error) {

	msg := c.NewMessage("blockchain", types.EventGetTransactionByHash, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetTxByHashes", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.TransactionDetails), nil
}

func (c *channelClient) GetMempool() (*types.ReplyTxList, error) {
	msg := c.NewMessage("mempool", types.EventGetMempool, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetMempool", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyTxList), nil
}

func (c *channelClient) GetAccounts() (*types.WalletAccounts, error) {
	msg := c.NewMessage("wallet", types.EventWalletGetAccountList, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetAccounts", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccounts), nil
}

func (c *channelClient) NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error) {
	msg := c.NewMessage("wallet", types.EventNewAccount, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("NewAccount", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccount), nil
}

func (c *channelClient) WalletTxList(parm *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	msg := c.NewMessage("wallet", types.EventWalletTransactionList, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("NewAccount", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletTxDetails), nil
}

func (c *channelClient) ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	msg := c.NewMessage("wallet", types.EventWalletImportprivkey, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("ImportPrivkey", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccount), nil
}

func (c *channelClient) SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	msg := c.NewMessage("wallet", types.EventWalletSendToAddress, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("SendToAddress", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHash), nil
}

func (c *channelClient) SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error) {
	msg := c.NewMessage("wallet", types.EventWalletSetFee, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("SetTxFee", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil

}

func (c *channelClient) SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	msg := c.NewMessage("wallet", types.EventWalletSetLabel, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("SetLabl", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.WalletAccount), nil
}

func (c *channelClient) MergeBalance(parm *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	msg := c.NewMessage("wallet", types.EventWalletMergeBalance, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("MergeBalance", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHashes), nil
}

func (c *channelClient) SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error) {
	msg := c.NewMessage("wallet", types.EventWalletSetPasswd, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("SetPasswd", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (c *channelClient) Lock() (*types.Reply, error) {
	msg := c.NewMessage("wallet", types.EventWalletLock, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("Lock", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (c *channelClient) UnLock(parm *types.WalletUnLock) (*types.Reply, error) {
	msg := c.NewMessage("wallet", types.EventWalletUnLock, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("UnLock", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (c *channelClient) GetPeerInfo() (*types.PeerList, error) {
	msg := c.NewMessage("p2p", types.EventPeerInfo, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetPeerInfo", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.PeerList), nil
}

func (c *channelClient) GetHeaders(in *types.ReqBlocks) (*types.Headers, error) {
	msg := c.NewMessage("blockchain", types.EventGetHeaders, &types.ReqBlocks{Start: in.GetStart(), End: in.GetEnd(),
		Isdetail: in.GetIsdetail()})
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetHeaders", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.Headers), nil
}

func (c *channelClient) GetLastMemPool(*types.ReqNil) (*types.ReplyTxList, error) {
	msg := c.NewMessage("mempool", types.EventGetLastMempool, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetLastMemPool", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplyTxList), nil
}

func (c *channelClient) GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error) {
	msg := c.NewMessage("blockchain", types.EventGetBlockOverview, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetBlockOverview", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.BlockOverview), nil
}

func (c *channelClient) GetAddrOverview(parm *types.ReqAddr) (*types.AddrOverview, error) {
	msg := c.NewMessage("blockchain", types.EventGetAddrOverview, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	addrOverview := resp.Data.(*types.AddrOverview)

	//获取地址账户的余额通过account模块
	addrs := make([]string, 1)
	addrs[0] = parm.Addr
	accounts, err := accountdb.LoadAccounts(c.Client, addrs)
	if err != nil {
		return nil, err
	}
	if len(accounts) != 0 {
		addrOverview.Balance = accounts[0].Balance
	}
	return addrOverview, nil
}

func (c *channelClient) GetBlockHash(parm *types.ReqInt) (*types.ReplyHash, error) {
	msg := c.NewMessage("blockchain", types.EventGetBlockHash, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetBlockHash", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHash), nil
}

//seed
func (c *channelClient) GenSeed(parm *types.GenSeedLang) (*types.ReplySeed, error) {
	msg := c.NewMessage("wallet", types.EventGenSeed, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GenSeed", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplySeed), nil
}

func (c *channelClient) SaveSeed(parm *types.SaveSeedByPw) (*types.Reply, error) {
	msg := c.NewMessage("wallet", types.EventSaveSeed, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("SaveSeed", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.Reply), nil
}
func (c *channelClient) GetSeed(parm *types.GetSeedByPw) (*types.ReplySeed, error) {
	msg := c.NewMessage("wallet", types.EventGetSeed, parm)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetSeed", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplySeed), nil
}

func (c *channelClient) GetWalletStatus() (*WalletStatus, error) {
	msg := c.NewMessage("wallet", types.EventGetWalletStatus, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("GetWalletStatus", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}

	return (*WalletStatus)(resp.Data.(*types.WalletStatus)), nil
}

func (c *channelClient) GetBalance(in *types.ReqBalance) ([]*types.Account, error) {

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

		accounts, err := accountdb.LoadAccounts(c.Client, exaddrs)
		if err != nil {
			log.Error("GetBalance", "err", err.Error())
			return nil, err
		}
		return accounts, nil
	default:
		execaddress := account.ExecAddress(in.GetExecer())
		addrs := in.GetAddresses()
		var accounts []*types.Account
		for _, addr := range addrs {

			account, err := accountdb.LoadExecAccountQueue(c.Client, addr, execaddress.String())
			if err != nil {
				log.Error("GetBalance", "err", err.Error())
				continue
			}
			accounts = append(accounts, account)
		}

		return accounts, nil
	}
	return nil, nil
}

func (c *channelClient) QueryHash(in *types.Query) (*types.Message, error) {

	msg := c.NewMessage("blockchain", types.EventQuery, in)
	err := c.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	querydata := resp.GetData().(types.Message)

	return &querydata, nil

}

func (c *channelClient) SetAutoMiner(in *types.MinerFlag) (*types.Reply, error) {

	msg := c.NewMessage("wallet", types.EventWalletAutoMiner, in)
	err := c.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.Reply), nil
}

func (c *channelClient) GetTicketCount() (*types.Int64, error) {
	msg := c.NewMessage("consensus", types.EventGetTicketCount, nil)
	err := c.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.Int64), nil
}

func (c *channelClient) DumpPrivkey(in *types.ReqStr) (*types.ReplyStr, error) {
	msg := c.NewMessage("wallet", types.EventDumpPrivkey, in)
	err := c.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.ReplyStr), nil
}

func (c *channelClient) CloseTickets() (*types.TxHashList, error) {
	msg := c.NewMessage("wallet", types.EventCloseTickets, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("CloseTickets", "Error", err.Error())
		return nil, err
	}
	resp, err := c.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.TxHashList), nil
}

func (c *channelClient) GetTotalCoins(in *types.ReqGetTotalCoins) (*types.ReplyGetTotalCoins, error) {
	//获取地址账户的余额通过account模块
	resp, err := accountdb.GetTotalCoins(c.Client, in)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *channelClient) IsSync() bool {
	msg := c.NewMessage("blockchain", types.EventIsSync, nil)
	err := c.Send(msg, true)
	if err != nil {
		log.Error("IsSync", "Send Error", err.Error())
		return false
	}

	resp, err := c.Wait(msg)
	if err != nil {
		log.Error("IsSync", "Wait Error", err.Error())
		return false
	}
	return resp.GetData().(*types.IsCaughtUp).GetIscaughtup()
}
