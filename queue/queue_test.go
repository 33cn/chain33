package queue

import (
	"log"
	"testing"

	"code.aliyun.com/chain33/chain33/types"
)

func TestMultiTopic(t *testing.T) {
	q := New("channel")

	//mempool
	go func() {
		client := q.GetClient()
		client.Sub("mempool")
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				log.Println("recv msg:", msg)
				msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{true, []byte("word")}))
			}
		}
	}()

	//blockchain
	go func() {
		client := q.GetClient()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			if msg.Ty == types.EventGetBlockHeight {
				msg.Reply(client.NewMessage("blockchain", types.EventReplyBlockHeight, types.ReplyBlockHeight{100}))
			}
		}
	}()

	//rpc server
	go func() {
		client := q.GetClient()
		//rpc 模块 会向其他模块发送消息，自己本身不需要订阅消息
		msg := client.NewMessage("mempool", types.EventTx, "hello")
		log.Println("send tx")
		client.Send(msg, true)
		log.Println("send tx ok ")
		reply, err := client.Wait(msg)
		if err != nil {
			t.Error(err)
			return
		}
		log.Println("wait message ok ")
		t.Log(string(reply.GetData().(types.Reply).Msg))

		msg = client.NewMessage("blockchain", types.EventGetBlockHeight, nil)
		client.Send(msg, true)
		reply, err = client.Wait(msg)
		if err != nil {
			t.Error(err)
			return
		}
		t.Log(reply)
		log.Println("close")
		q.Close()
	}()
	log.Println("start")
	q.Start()
}
