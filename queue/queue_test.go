package queue

import (
	"log"
	"testing"
	"time"

	"gitlab.33.cn/chain33/chain33/types"
)

func init() {
	DisableLog()
}

func TestMultiTopic(t *testing.T) {
	q := New("channel")

	//mempool
	go func() {
		client := q.Client()
		client.Sub("mempool")
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{IsOk: true, Msg: []byte("word")}))
			}
		}
	}()

	//blockchain
	go func() {
		client := q.Client()
		client.Sub("blockchain")
		for msg := range client.Recv() {
			if msg.Ty == types.EventGetBlockHeight {
				msg.Reply(client.NewMessage("blockchain", types.EventReplyBlockHeight, types.ReplyBlockHeight{Height: 100}))
			}
		}
	}()

	//rpc server
	go func() {
		client := q.Client()
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
		client := q.Client()
		client.Sub("mempool")
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				time.Sleep(time.Second)
				msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{IsOk: true, Msg: []byte("word")}))
			}
		}
	}()

	//rpc server
	go func() {
		client := q.Client()
		//rpc 模块 会向其他模块发送消息，自己本身不需要订阅消息
		for {
			msg := client.NewMessage("mempool", types.EventTx, "hello")
			err := client.Send(msg, false)
			if err != nil {
				break
			}
		}
		//high 优先级
		msg := client.NewMessage("mempool", types.EventTx, "hello")
		client.Send(msg, true)
		reply, err := client.Wait(msg)
		if err != nil {
			t.Error(err)
			return
		}
		t.Log(string(reply.GetData().(types.Reply).Msg))
		q.Close()
	}()
	log.Println("start")
	q.Start()
}

//发送100000 低优先级的消息，然后发送一个高优先级的消息
//高优先级的消息可以即时返回
func TestClientClose(t *testing.T) {
	q := New("channel")
	//mempool
	go func() {
		client := q.Client()
		client.Sub("mempool")
		i := 0
		for msg := range client.Recv() {
			if msg.Ty == types.EventTx {
				time.Sleep(time.Second / 10)
				msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{IsOk: true, Msg: []byte("word")}))
			}
			i++
			if i == 10 {
				go func() {
					client.Close()
					qlog.Info("close ok")
				}()
			}
		}
	}()

	//rpc server
	go func() {
		client := q.Client()
		//high 优先级
		done := make(chan struct{}, 100)
		for i := 0; i < 100; i++ {
			go func() {
				defer func() {
					done <- struct{}{}
				}()
				msg := client.NewMessage("mempool", types.EventTx, "hello")
				err := client.Send(msg, true)
				if err != nil { //chan is closed
					log.Println(err)
					return
				}
				_, err = client.Wait(msg)
				if err != nil {
					t.Error(err)
					return
				}
			}()
		}
		for i := 0; i < 100; i++ {
			<-done
		}
		q.Close()
	}()
	q.Start()
}

func TestPrintMessage(t *testing.T) {
	q := New("channel")
	client := q.Client()
	msg := client.NewMessage("mempool", types.EventReply, types.Reply{IsOk: true, Msg: []byte("word")})
	t.Log(msg)
}
