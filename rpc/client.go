package rpc

import (
	"code.aliyun.com/chain33/chain33/types"
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
	client.qclient.Sub("rpc")
	go func() {
		for msg := range client.qclient.Recv() {

			msgty := msg.Ty
			switch msgty {
			case types.EventTx:

				resp := client.SendTx(msg.GetData().(*types.Transaction))
				msg.Reply(client.qclient.NewMessage("channel", types.EventTx, resp.GetData()))
			case types.EventQueryTx:

				proof, err := client.QueryTx(msg.GetData().([]byte))
				if err != nil {
					msg.Reply(client.qclient.NewMessage("channel", types.EventTx, types.Reply{false, []byte(err.Error())}))
					return
				}
				msg.Reply(client.qclient.NewMessage("channel", types.EventQueryTx, proof))
			case types.EventGetBlocks:

				blocks, err := client.GetBlocks(msg.GetData().(types.RequestBlocks).Start, msg.Data.(types.RequestBlocks).End)
				if err != nil {
					msg.Reply(client.qclient.NewMessage("channel", types.EventTx, types.Reply{false, []byte(err.Error())}))
					return
				}
				msg.Reply(client.qclient.NewMessage("channel", types.EventGetBlocks, blocks))

			default:

				msg.Reply(client.qclient.NewMessage("channel", types.EventTx, types.Reply{false, []byte("wrong type")}))

			}
		}
	}()

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
		return resp
	}

	return resp
}

func (client *channelClient) GetBlocks(start int64, end int64) (blocks *types.Blocks, err error) {
	msg := client.qclient.NewMessage("blockchain", types.EventGetBlocks, &types.RequestBlocks{start, end})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.Blocks), nil
}

func (client *channelClient) QueryTx(hash []byte) (proof *types.MerkleProof, err error) {
	msg := client.qclient.NewMessage("blockchain", types.EventQueryTx, &types.RequestHash{hash})
	client.qclient.Send(msg, true)
	resp, err := client.qclient.Wait(msg)
	if err != nil {
		return nil, err
	}
	if resp.Err() != nil {
		return nil, resp.Err()
	}
	return resp.Data.(*types.MerkleProof), nil
}
