package rpc

import (
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/types"
)

//提供系统rpc接口
//sendTx
//status
//channel 主要用于内部测试，实际情况主要采用 jsonrpc 和 grpc

type IRClient interface {
	SendTx(tx *types.Transaction) queue.Message
	SetQueue(q *queue.Queue)
	QueryTx(hash []byte) (proof *types.TransactionDetail, err error)
	GetBlocks(start int64, end int64, isdetail bool) (blocks *types.BlockDetails, err error)
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
