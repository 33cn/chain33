package main

import (
	"log"
	"queue"
	"rpc"

	"code.aliyun.com/chain33/chain33/types"
)

func main() {

	q := queue.New("channel")
	api := rpc.NewServer("jsonrpc", ":8801")
	api.SetQueue(q)
	gapi := rpc.NewServer("grpc", ":8802")
	gapi.SetQueue(q)
	go func(Q *queue.Queue) {
		//jsonrpc, grpc, channel 三种模式
		client := rpc.NewClient("channel", "")
		//同步接口
		client.SetQueue(q)

	}(q)

	go func() {
		client := q.GetClient()
		client.Sub("mempool")
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				log.Println("recv msg:", msg)
				msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{true, []byte("you success")}))
			}
		}
	}()

	q.Start()
}
