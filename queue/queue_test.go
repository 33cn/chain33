package queue

import (
	"log"
	"testing"
	"time"

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

//发送100000 低优先级的消息，然后发送一个高优先级的消息
//高优先级的消息可以即时返回
func TestHighLow(t *testing.T) {
	q := New("channel")

	//mempool
	go func() {
		client := q.GetClient()
		client.Sub("mempool")
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				time.Sleep(time.Second)
				log.Println("recv msg:", msg)
				msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{true, []byte("word")}))
			}
		}
	}()

	//rpc server
	go func() {
		client := q.GetClient()
		//rpc 模块 会向其他模块发送消息，自己本身不需要订阅消息
		for {
			msg := client.NewMessage("mempool", types.EventTx, "hello")
			err := client.Send(msg, false)
			if err != nil {
				break
			}
		}
		//high 优先级
		log.Println("send high tx")
		msg := client.NewMessage("mempool", types.EventTx, "hello")
		client.Send(msg, true)
		reply, err := client.Wait(msg)
		if err != nil {
			t.Error(err)
			return
		}
		log.Println("wait message ok ")
		t.Log(string(reply.GetData().(types.Reply).Msg))
		q.Close()
	}()
	log.Println("start")
	q.Start()
}

func TestPrintMessage(t *testing.T) {
	q := New("channel")
	client := q.GetClient()
	msg := client.NewMessage("mempool", types.EventReply, types.Reply{true, []byte("word")})
	t.Log(msg)
}
