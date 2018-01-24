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
}

type channelClient struct {
	qclient queue.IClient
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

	client.qclient = q.GetClient()
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
	parm.Tx.Signature = &types.Signature{parm.GetTy(), parm.GetPubkey(), parm.GetSign()}
	msg := client.qclient.NewMessage("mempool", types.EventTx, parm.GetTx())
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		resp.Data = err
	}
	return resp

}

//channel
func (client *channelClient) SendTx(tx *types.Transaction) queue.Message {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTx, tx)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {

		resp.Data = err
	}

	return resp
}

func (client *channelClient) GetBlocks(start int64, end int64, isdetail bool) (blocks *types.BlockDetails, err error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlocks, &types.ReqBlocks{start, end, isdetail})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.BlockDetails), nil
}

func (client *channelClient) QueryTx(hash []byte) (proof *types.TransactionDetail, err error) {
	msg := client.qclient.NewMessage("blockchain", types.EventQueryTx, &types.ReqHash{hash})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.TransactionDetail), nil
}

func (client *channelClient) GetLastHeader() (*types.Header, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetLastHeader, nil)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Header), nil
}

func (client *channelClient) GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfos, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetTransactionByAddr, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyTxInfos), nil
}

func (client *channelClient) GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error) {

	msg := client.qclient.NewMessage("blockchain", types.EventGetTransactionByHash, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.TransactionDetails), nil
}

func (client *channelClient) GetMempool() (*types.ReplyTxList, error) {
	msg := client.qclient.NewMessage("mempool", types.EventGetMempool, nil)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyTxList), nil
}

func (client *channelClient) GetAccounts() (*types.WalletAccounts, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletGetAccountList, nil)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccounts), nil
}

func (client *channelClient) NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error) {
	msg := client.qclient.NewMessage("wallet", types.EventNewAccount, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccount), nil
}

func (client *channelClient) WalletTxList(parm *types.ReqWalletTransactionList) (*types.WalletTxDetails, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletTransactionList, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletTxDetails), nil
}

func (client *channelClient) ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletImportprivkey, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.WalletAccount), nil
}

func (client *channelClient) SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSendToAddress, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHash), nil
}

func (client *channelClient) SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSetFee, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil

}

func (client *channelClient) SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSetLabel, parm)
	client.qclient.Send(msg, true)
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
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHashes), nil
}

func (client *channelClient) SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSetPasswd, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (client *channelClient) Lock() (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletLock, nil)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (client *channelClient) UnLock(parm *types.WalletUnLock) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletUnLock, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.Reply), nil
}

func (client *channelClient) GetPeerInfo() (*types.PeerList, error) {
	msg := client.qclient.NewMessage("p2p", types.EventPeerInfo, nil)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.PeerList), nil
}

func (client *channelClient) GetHeaders(in *types.ReqBlocks) (*types.Headers, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetHeaders, &types.ReqBlocks{Start: in.GetStart(), End: in.GetEnd(),
		Isdetail: in.GetIsdetail()})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.Headers), nil
}

func (client *channelClient) GetLastMemPool(*types.ReqNil) (*types.ReplyTxList, error) {
	msg := client.qclient.NewMessage("mempool", types.EventGetLastMempool, nil)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplyTxList), nil
}

func (client *channelClient) GetBlockOverview(parm *types.ReqHash) (*types.BlockOverview, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlockOverview, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.BlockOverview), nil
}

func (client *channelClient) GetAddrOverview(parm *types.ReqAddr) (*types.AddrOverview, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetAddrOverview, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	addrOverview := resp.Data.(*types.AddrOverview)

	//获取地址账户的余额通过account模块
	addrs := make([]string, 1)
	addrs[0] = parm.Addr
	accounts, err := account.LoadAccounts(client.q, addrs)
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
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}

	return resp.Data.(*types.ReplyHash), nil
}

//seed
func (client *channelClient) GenSeed(parm *types.GenSeedLang) (*types.ReplySeed, error) {
	msg := client.qclient.NewMessage("wallet", types.EventGenSeed, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplySeed), nil
}
func (client *channelClient) SaveSeed(parm *types.SaveSeedByPw) (*types.Reply, error) {
	msg := client.qclient.NewMessage("wallet", types.EventSaveSeed, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.Reply), nil
}
func (client *channelClient) GetSeed(parm *types.GetSeedByPw) (*types.ReplySeed, error) {
	msg := client.qclient.NewMessage("wallet", types.EventGetSeed, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	return resp.Data.(*types.ReplySeed), nil
}
