package rpc

import "code.aliyun.com/chain33/chain33/types"
import "code.aliyun.com/chain33/chain33/queue"

//提供系统rpc接口
//sendTx
//status
//channel 主要用于内部测试，实际情况主要采用 jsonrpc 和 grpc

type IClient interface {
	SendTx(tx *types.Transaction) (err error)
	SetQueue(q *queue.Queue)
	QueryTx(hash []byte) (proof *types.MerkleProof, err error)
	GetBlocks(start int64, end int64) (blocks *types.Blocks, err error)
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

func NewClient(name string, addr string) IClient {
	if name == "channel" {
		return &channelClient{}
	} else if name == "jsonrpc" {
		return &jsonClient{} //需要设置服务地址
	} else if name == "grpc" {
		return &grpcClient{} //需要设置服务地址
	}
	panic("client name not support")
}

func (client *channelClient) SetQueue(q *queue.Queue) {
	client.qclient = q.GetClient()
}

//channel
func (client *channelClient) SendTx(tx *types.Transaction) (err error) {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTx, 0, tx)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg.Id)
	if err != nil {
		return err
	}
	return resp.Err()
}

func (client *channelClient) GetBlocks(start int64, end int64) (blocks *types.Blocks, err error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlocks, 0, &types.RequestBlocks{start, end})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg.Id)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.Blocks), nil
}

func (client *channelClient) QueryTx(hash []byte) (proof *types.MerkleProof, err error) {
	msg := client.qclient.NewMessage("blockchain", types.EventQueryTx, 0, &types.RequestHash{hash})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg.Id)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.MerkleProof), nil
}
