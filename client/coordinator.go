// 封装系统模块间消息调用、协调模块
package client

import (
	"gitlab.33.cn/chain33/chain33/queue"
	"gitlab.33.cn/chain33/chain33/types"
)

// 消息通道交互API接口定义
type QueueProtocolAPI interface {
	// 同步发送交易信息到指定模块，获取应答消息
	QueryTx(data interface{}) (*types.Reply, error)
	QueryTxList(count int64, hashes [][]byte) (*types.ReplyTxList, error)

	QueryGetBlocksRange(start int64, end int64) (*types.BlockDetails, error)
	QueryGetBlocks(data interface{}) (*types.BlockDetails, error)
}

const (
	mempoolKey 		= "mempool"		// 未打包交易池
	p2pKey			= "p2p"			//
	rpcKey			= "rpc"
	consensusKey	= "consensus"	// 共识系统
	accountKey		= "accout"		// 账号系统
	executorKey		= "execs"		// 交易执行器
	walletKey		= "wallet"		// 钱包
	blockchainKey	= "blockchain"	// 区块
	storeKey		= "store"
)

// 消息通道协议实现
type QueueCoordinator struct {
	client		queue.Client	// 消息队列
}

func New() *QueueCoordinator {
	q := &QueueCoordinator{}

	return q
}

func (q *QueueCoordinator) SetQueueClient(client queue.Client)  {
	q.client = client
}

func (q *QueueCoordinator) Close()  {
	q.client.Close()
}

func (q *QueueCoordinator) query(topic string, ty int64,  data interface{}) (queue.Message, error) {
	msg := q.client.NewMessage(topic, ty, data)
	q.client.Send(msg, true)
	return q.client.Wait(msg)
}

func (q *QueueCoordinator) QueryTx(data interface{}) (*types.Reply, error) {
	msg, err := q.query(mempoolKey, types.EventTx, data)
	if nil!=err {
		return nil, err
	}
	return msg.GetData().(*types.Reply), nil
}

func (q *QueueCoordinator) QueryTxList(count int64, hashes [][]byte) (*types.ReplyTxList, error) {
	msg, err := q.query(mempoolKey, types.EventTxList, &types.TxHashList{Count: count, Hashes: hashes})
	if nil!=err {
		return nil, err
	}
	return msg.GetData().(*types.ReplyTxList), nil
}

func (q *QueueCoordinator) QueryGetBlocksRange(start int64, end int64) (*types.BlockDetails, error) {
	return q.QueryGetBlocks(&types.ReqBlocks{start,
	end,
	false,
	[]string{""}})
}

func (q *QueueCoordinator) QueryGetBlocks(data interface{}) (*types.BlockDetails, error) {
	msg, err := q.query(blockchainKey, types.EventGetBlocks, data)
	if nil!=err {
		return nil, err
	}
	return msg.GetData().(*types.BlockDetails), nil
}