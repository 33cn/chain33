package main

import (
	"code.aliyun.com/chain33/chain33/blockchain"
	"code.aliyun.com/chain33/chain33/consense"
	"code.aliyun.com/chain33/chain33/mempool"
	"code.aliyun.com/chain33/chain33/p2p"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/rpc"
)

func main() {
	q := queue.New()

	chain := blockchain.New()
	chain.SetQueue(q)

	con := consense.New("raft")
	con.SetQueue(q)

	mem := mempool.New()
	mem.SetQueue(q)

	network := p2p.New()
	network.SetQueue(q)

	api := rpc.New()
	api.SetQueue(q)

	go func() {
		client := rpc.NewClient()
		//同步接口
		client.SendTx("hello")
	}()
	q.Start()
}
