package rpc

import (
	"fmt"
	"types"
)
import "queue"

//提供系统rpc接口
//sendTx
//status
//channel 主要用于内部测试，实际情况主要采用 jsonrpc 和 grpc

type IRClient interface {
	SendTx(tx *types.Transaction) queue.Message
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
	go func() {
		for msg := range client.qclient.Recv() {

			msgty := msg.Ty
			switch msgty {
			case types.EventTx:

				//TODO:
				//单独测试的时候，可以把此行代码注释掉，模拟数据直接返回
				//var tr types.Transaction
				//tr.Account = []byte(msg.Data.(Pdata).TO)
				//resp := client.SendTx(&tr)
				msg.Data = "测试交易成功"
				resp := msg
				resp.ParentId = msg.Id
				client.qclient.Send(resp, false)
			case types.EventQueryTx:
				//TODO：
				msg.Data = "测试查询交易信息成功"
				msg.ParentId = msg.Id
				client.qclient.Send(msg, false)
			case types.EventGetBlocks:
				//TODO：
				msg.Data = "测试查询区块信息成功"
				msg.ParentId = msg.Id
				client.qclient.Send(msg, false)

			default:
				msg.Data = fmt.Errorf("wrong type")
				msg.ParentId = msg.Id
				client.qclient.Send(msg, false)

			}
		}
	}()

}

//channel
func (client *channelClient) SendTx(tx *types.Transaction) queue.Message {
	if client.qclient == nil {
		panic("client not bind message queue.")
	}
	msg := client.qclient.NewMessage("mempool", types.EventTx, 0, tx)
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg.Id)
	if err != nil {
		//	resp.Err().
		return resp
	}

	return resp
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
