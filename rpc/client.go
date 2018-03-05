package rpc

import (
	"errors"
	"math/rand"
	"time"
	//"unsafe"

	"code.aliyun.com/chain33/chain33/account"

	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

//提供系统rpc接口
//sendTx
//status
//channel 主要用于内部测试，实际情况主要采用 jsonrpc 和 Grpc

type IRClient interface {
	SendTx(tx *types.Transaction) queue.Message
	CreateRawTransaction(parm *types.CreateTx) ([]byte, error)
	SendRawTransaction(parm *types.SignedTx) queue.Message
	SetQueue(q *queue.Queue)
	QueryTx(hash []byte) (proof *types.TransactionDetail, err error)
	GetBlocks(start int64, end int64, isdetail bool) (blocks *types.BlockDetails, err error)
	GetLastHeader() (*types.Header, error)
	GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfos, error)
	GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error)
	GetMempool() (*types.ReplyTxList, error)
	GetAccounts() (*types.WalletAccounts, error)
	NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error)
	WalletTxList(parm *types.ReqWalletTransactionList) (*types.WalletTxDetails, error)
	ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error)
	SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error)
	SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error)
	SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error)
	MergeBalance(parm *types.ReqWalletMergeBalance) (*types.ReplyHashes, error)
	SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error)
	Lock() (*types.Reply, error)
	UnLock(parm *types.WalletUnLock) (*types.Reply, error)
	GetPeerInfo() (*types.PeerList, error)
	GetHeaders(*types.ReqBlocks) (*types.Headers, error)
	GetLastMemPool(*types.ReqNil) (*types.ReplyTxList, error)

	GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error)
	GetAddrOverview(parm *types.ReqAddr) (*types.AddrOverview, error)
	GetBlockHash(parm *types.ReqInt) (*types.ReplyHash, error)
	//seed
	GenSeed(parm *types.GenSeedLang) (*types.ReplySeed, error)
	GetSeed(parm *types.GetSeedByPw) (*types.ReplySeed, error)
	SaveSeed(parm *types.SaveSeedByPw) (*types.Reply, error)
	GetWalletStatus() (*WalletStatus, error)
	//getbalance
	GetBalance(*types.ReqBalance) ([]*types.Account, error)
	//query
	QueryHash(*types.Query) (*types.Message, error)
	//miner
	SetAutoMiner(*types.MinerFlag) (*types.Reply, error)
	GetTicketCount() (*types.Int64, error)
	DumpPrivkey(*types.ReqStr) (*types.ReplyStr, error)
}

type channelClient struct {
	qclient queue.Client
	q       *queue.Queue
}

type jsonClient struct {
	channelClient
}

type grpcClient struct {
	channelClient
}

func NewClient(name string, addr string) IRClient {
	if name == "channel" {
		return &channelClient{}
	} else if name == "jsonrpc" {
		return &jsonClient{} //需要设置服务地址，与其他模块通信使用
	} else if name == "grpc" {
		return &grpcClient{} //需要设置服务地址，与其他模块通信使用
	}
	panic("client name not support")
}

func (client *channelClient) SetQueue(q *queue.Queue) {

	client.qclient = q.NewClient()
	client.q = q

}

func (client *channelClient) CreateRawTransaction(parm *types.CreateTx) ([]byte, error) {
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

func (client *channelClient) SendRawTransaction(parm *types.SignedTx) queue.Message {
	var tx types.Transaction
	err := types.Decode(parm.GetUnsign(), &tx)

	if err == nil {
		tx.Signature = &types.Signature{parm.GetTy(), parm.GetPubkey(), parm.GetSign()}
		msg := client.qclient.NewMessage("mempool", types.EventTx, &tx)
		err := client.qclient.Send(msg, true)
		if err != nil {
			var msg queue.Message
			log.Error("SendRawTransaction", "Error", err.Error())
			msg.Data = err
			return msg
		}
		resp, err := client.qclient.Wait(msg)

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
func (client *channelClient) SendTx(tx *types.Transaction) queue.Message {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTx, tx)
	err := client.qclient.Send(msg, true)
	if err != nil {
		var msg queue.Message
		log.Error("SendTx", "Error", err.Error())
		msg.Data = err
		return msg
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {

		resp.Data = err
	}
	if resp.GetData().(*types.Reply).GetIsOk() {
		resp.GetData().(*types.Reply).Msg = tx.Hash()
	}
	return resp
}

func (client *channelClient) GetBlocks(start int64, end int64, isdetail bool) (*types.BlockDetails, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, end, isdetail, []string{""}})
	err := client.qclient.Send(msg, true)
	if err != nil {

		log.Error("SendRawTransaction", "Error", err.Error())

		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.BlockDetails), nil
}

func (client *channelClient) QueryTx(hash []byte) (*types.TransactionDetail, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventQueryTx, &types.ReqHash{hash})
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("QueryTx", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.TransactionDetail), nil
}

func (client *channelClient) GetLastHeader() (*types.Header, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetLastHeader, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetLastHeader", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Header), nil
}

func (client *channelClient) GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfos, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetTransactionByAddr, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetTxByAddr", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyTxInfos), nil
}

func (client *channelClient) GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error) {

	msg := client.qclient.NewMessage("blockchain", types.EventGetTransactionByHash, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetTxByHashes", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.TransactionDetails), nil
}

func (client *channelClient) GetMempool() (*types.ReplyTxList, error) {
	msg := client.qclient.NewMessage("mempool", types.EventGetMempool, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetMempool", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyTxList), nil
}

func (client *channelClient) GetAccounts() (*types.WalletAccounts, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletGetAccountList, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetAccounts", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccounts), nil
}

func (client *channelClient) NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error) {
	msg := client.qclient.NewMessage("wallet", types.EventNewAccount, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("NewAccount", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccount), nil
}

func (client *channelClient) WalletTxList(parm *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletTransactionList, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("NewAccount", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletTxDetails), nil
}

func (client *channelClient) ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletImportprivkey, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("ImportPrivkey", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccount), nil
}

func (client *channelClient) SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSendToAddress, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("SendToAddress", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHash), nil
}

func (client *channelClient) SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSetFee, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("SetTxFee", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil

}

func (client *channelClient) SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSetLabel, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("SetLabl", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.WalletAccount), nil
}

func (client *channelClient) MergeBalance(parm *types.ReqWalletMergeBalance) (*types.ReplyHashes, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletMergeBalance, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("MergeBalance", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHashes), nil
}

func (client *channelClient) SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSetPasswd, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("SetPasswd", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (client *channelClient) Lock() (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletLock, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("Lock", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (client *channelClient) UnLock(parm *types.WalletUnLock) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletUnLock, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("UnLock", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (client *channelClient) GetPeerInfo() (*types.PeerList, error) {
	msg := client.qclient.NewMessage("p2p", types.EventPeerInfo, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetPeerInfo", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.PeerList), nil
}

func (client *channelClient) GetHeaders(in *types.ReqBlocks) (*types.Headers, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetHeaders, &types.ReqBlocks{Start: in.GetStart(), End: in.GetEnd(),
		Isdetail: in.GetIsdetail()})
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetHeaders", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.Headers), nil
}

func (client *channelClient) GetLastMemPool(*types.ReqNil) (*types.ReplyTxList, error) {
	msg := client.qclient.NewMessage("mempool", types.EventGetLastMempool, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetLastMemPool", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplyTxList), nil
}

func (client *channelClient) GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlockOverview, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetBlockOverview", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.BlockOverview), nil
}

func (client *channelClient) GetAddrOverview(parm *types.ReqAddr) (*types.AddrOverview, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetAddrOverview, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetAddrOverview", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	addrOverview := resp.Data.(*types.AddrOverview)

	//获取地址账户的余额通过account模块
	addrs := make([]string, 1)
	addrs[0] = parm.Addr
	accounts, err := account.LoadAccounts(client.qclient, addrs)
	if err != nil {
		return nil, err
	}
	if len(accounts) != 0 {
		addrOverview.Balance = accounts[0].Balance
	}
	return addrOverview, nil
}

func (client *channelClient) GetBlockHash(parm *types.ReqInt) (*types.ReplyHash, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlockHash, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetBlockHash", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHash), nil
}

//seed
func (client *channelClient) GenSeed(parm *types.GenSeedLang) (*types.ReplySeed, error) {
	msg := client.qclient.NewMessage("wallet", types.EventGenSeed, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GenSeed", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplySeed), nil
}

func (client *channelClient) SaveSeed(parm *types.SaveSeedByPw) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventSaveSeed, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("SaveSeed", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.Reply), nil
}
func (client *channelClient) GetSeed(parm *types.GetSeedByPw) (*types.ReplySeed, error) {
	msg := client.qclient.NewMessage("wallet", types.EventGetSeed, parm)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetSeed", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplySeed), nil
}

func (client *channelClient) GetWalletStatus() (*WalletStatus, error) {
	msg := client.qclient.NewMessage("wallet", types.EventGetWalletStatus, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		log.Error("GetWalletStatus", "Error", err.Error())
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return (*WalletStatus)(resp.Data.(*types.WalletStatus)), nil
}

func (client *channelClient) GetBalance(in *types.ReqBalance) ([]*types.Account, error) {

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
		accounts, err := account.LoadAccounts(client.qclient, exaddrs)
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
			account, err := account.LoadExecAccountQueue(client.qclient, addr, execaddress.String())
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

func (client *channelClient) QueryHash(in *types.Query) (*types.Message, error) {

	msg := client.qclient.NewMessage("blockchain", types.EventQuery, in)
	err := client.qclient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	querydata := resp.GetData().(types.Message)

	return &querydata, nil

}

func (client *channelClient) SetAutoMiner(in *types.MinerFlag) (*types.Reply, error) {

	msg := client.qclient.NewMessage("wallet", types.EventWalletAutoMiner, in)
	err := client.qclient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.Reply), nil
}

func (client *channelClient) GetTicketCount() (*types.Int64, error) {
	msg := client.qclient.NewMessage("consensus", types.EventGetTicketCount, nil)
	err := client.qclient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.Int64), nil
}

func (client *channelClient) DumpPrivkey(in *types.ReqStr) (*types.ReplyStr, error) {
	msg := client.qclient.NewMessage("wallet", types.EventDumpPrivkey, in)
	err := client.qclient.Send(msg, true)
	if err != nil {
		return nil, err
	}
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.GetData().(*types.ReplyStr), nil
}
