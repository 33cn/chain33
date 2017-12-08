package rpc

import (
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

//提供系统rpc接口
//sendTx
//status
//channel 主要用于内部测试，实际情况主要采用 jsonrpc 和 Grpc

type IRClient interface {
	SendTx(tx *types.Transaction) queue.Message
	SetQueue(q *queue.Queue)
	QueryTx(hash []byte) (proof *types.TransactionDetail, err error)
	GetBlocks(start int64, end int64, isdetail bool) (blocks *types.BlockDetails, err error)
	GetLastHeader() (*types.Header, error)
	GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfo, error)
	GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error)
	GetMempool() (*types.ReplyTxList, error)
	GetAccounts() (*types.WalletAccounts, error)
	NewAccount(parm *types.ReqNewAccount) (*types.WalletAccount, error)
	WalletTxList(parm *types.ReqWalletTransactionList) (*types.TransactionDetails, error)
	ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error)
	SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error)
	SetTxFee(parm *types.ReqWalletSetFee) (*types.Reply, error)
	SetLabl(parm *types.ReqWalletSetLabel) (*types.WalletAccount, error)
	MergeBalance(parm *types.ReqWalletMergeBalance) (*types.ReplyHashes, error)
	SetPasswd(parm *types.ReqWalletSetPasswd) (*types.Reply, error)
	Lock() (*types.Reply, error)
	UnLock(parm *types.WalletUnLock) (*types.Reply, error)
	GetPeerInfo() (*types.PeerList, error)
}

type channelClient struct {
	qclient queue.IClient
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.Header), nil
}

func (client *channelClient) GetTxByAddr(parm *types.ReqAddr) (*types.ReplyTxInfo, error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetTransactionByAddr, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.ReplyTxInfo), nil
}

func (client *channelClient) GetTxByHashes(parm *types.ReqHashes) (*types.TransactionDetails, error) {

	msg := client.qclient.NewMessage("blockchain", types.EventGetTransactionByHash, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.WalletAccount), nil
}

func (client *channelClient) WalletTxList(parm *types.ReqWalletTransactionList) (*types.TransactionDetails, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletTransactionList, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.TransactionDetails), nil
}

func (client *channelClient) ImportPrivkey(parm *types.ReqWalletImportPrivKey) (*types.WalletAccount, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletImportprivkey, parm)
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

func (client *channelClient) SendToAddress(parm *types.ReqWalletSendToAddress) (*types.ReplyHash, error) {
	msg := client.qclient.NewMessage("wallet", types.EventWalletSendToAddress, parm)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
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
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.PeerList), nil
}
