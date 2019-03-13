// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"fmt"
	"testing"
	"time"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func init() {
	DisableLog()
}

func TestTimeout(t *testing.T) {
	//send timeout and recv timeout
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
	client := q.Client()
	//rpc 模块 会向其他模块发送消息，自己本身不需要订阅消息
	msg := client.NewMessage("blockchain", types.EventTx, "hello")
	for i := 0; i < defaultChanBuffer; i++ {
		err := client.SendTimeout(msg, true, 0)
		if err != nil {
			t.Error(err)
			return
		}
	}

	//再发送一个交易返回chain full
	err := client.SendTimeout(msg, true, 0)
	if err != ErrQueueChannelFull {
		t.Error(err)
		return
	}

	//发送一个交易返回返回timeout
	err = client.SendTimeout(msg, true, time.Millisecond)
	if err != ErrQueueTimeout {
		t.Error(err)
		return
	}
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
		client.Send(msg, true)
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
		q.Close()
	}()
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
			err := client.SendTimeout(msg, false, 0)
			if err != nil {
				fmt.Println(err)
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
				if err == types.ErrChannelClosed {
					return
				}
				if err != nil { //chan is closed
					t.Error(err)
					return
				}
				_, err = client.Wait(msg)
				if err == types.ErrChannelClosed {
					return
				}
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

func TestMessage_ReplyErr(t *testing.T) {
	q := New("channel")
	assert.Equal(t, "channel", q.Name())
	//接收消息
	go func() {
		client := q.Client()
		client.Sub("mempool")
		for msg := range client.Recv() {
			if msg.Data == nil {
				msg.ReplyErr("test", fmt.Errorf("test error"))
				break
			}
			msg.Reply(NewMessage(0, "mempool", types.EventReply, types.Reply{IsOk: true, Msg: []byte("test ok")}))
		}
	}()

	//发送消息
	go func() {
		client := q.Client()
		msg := client.NewMessage("mempool", types.EventTx, "hello")
		err := client.Send(msg, true)
		if err != nil { //chan is closed
			t.Error(err)
			return
		}

		msg = client.NewMessage("mempool", types.EventTx, nil)
		err = client.Send(msg, true)
		if err != nil {
			t.Error(err)
			return
		}
		client.CloseQueue()
	}()

	q.Start()
}

func TestChanSubCallback(t *testing.T) {
	q := New("channel")
	client := q.Client()
	client.Sub("hello")
	done := make(chan struct{}, 1025)
	go func() {
		for i := 0; i < 1025; i++ {
			sub := q.(*queue).chanSub("hello")
			msg := NewMessageCallback(1, "", 0, nil, func(msg *Message) {
				done <- struct{}{}
			})
			sub.high <- msg
		}
	}()
	for i := 0; i < 1025; i++ {
		msg := <-client.Recv()
		client.Reply(msg)
	}
	for i := 0; i < 1025; i++ {
		<-done
	}
}

func BenchmarkSendMessage(b *testing.B) {
	q := New("channel")
	//mempool
	b.ReportAllocs()
	go func() {
		client := q.Client()
		client.Sub("mempool")
		defer client.Close()
		for msg := range client.Recv() {
			go func(msg *Message) {
				if msg.Ty == types.EventTx {
					msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{IsOk: true, Msg: []byte("word")}))
				}
			}(msg)
		}
	}()
	go q.Start()
	client := q.Client()
	//high 优先级
	msg := client.NewMessage("mempool", types.EventTx, "hello")
	for i := 0; i < b.N; i++ {
		err := client.Send(msg, true)
		if err != nil {
			b.Error(err)
			return
		}
		_, err = client.Wait(msg)
		if err != nil {
			b.Error(err)
			return
		}
	}
}

func BenchmarkStructChan(b *testing.B) {
	ch := make(chan struct{})
	go func() {
		for {
			<-ch
		}
	}()
	for i := 0; i < b.N; i++ {
		ch <- struct{}{}
	}
}

func BenchmarkBoolChan(b *testing.B) {
	ch := make(chan bool)
	go func() {
		for {
			<-ch
		}
	}()
	for i := 0; i < b.N; i++ {
		ch <- true
	}
}

func BenchmarkIntChan(b *testing.B) {
	ch := make(chan int)
	go func() {
		for {
			<-ch
		}
	}()
	for i := 0; i < b.N; i++ {
		ch <- 1
	}
}

func BenchmarkChanSub(b *testing.B) {
	q := New("channel")
	done := make(chan struct{})
	go func() {
		for i := 0; i < b.N; i++ {
			q.(*queue).chanSub("hello")
		}
		done <- struct{}{}
	}()
	for i := 0; i < b.N; i++ {
		q.(*queue).chanSub("hello")
	}
	<-done
}

func BenchmarkChanSub2(b *testing.B) {
	q := New("channel")
	client := q.Client()
	client.Sub("hello")

	go func() {
		for i := 0; i < b.N; i++ {
			sub := q.(*queue).chanSub("hello")
			msg := NewMessage(1, "", 0, nil)
			sub.high <- msg
			_, err := client.Wait(msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	}()
	for i := 0; i < b.N; i++ {
		msg := <-client.Recv()
		msg.Reply(msg)
	}
}

func BenchmarkChanSub3(b *testing.B) {
	q := New("channel")
	client := q.Client()
	client.Sub("hello")

	go func() {
		for i := 0; i < b.N; i++ {
			sub := q.(*queue).chanSub("hello")
			msg := NewMessage(1, "", 0, nil)
			sub.high <- msg
		}
	}()
	for i := 0; i < b.N; i++ {
		msg := <-client.Recv()
		msg.Reply(msg)
	}
}

func BenchmarkChanSub4(b *testing.B) {
	q := New("channel")
	client := q.Client()
	client.Sub("hello")

	go func() {
		for i := 0; i < b.N; i++ {
			sub := q.(*queue).chanSub("hello")
			msg := &Message{ID: 1}
			sub.high <- msg
		}
	}()
	for i := 0; i < b.N; i++ {
		<-client.Recv()
	}
}

func BenchmarkChanSubCallback(b *testing.B) {
	q := New("channel")
	client := q.Client()
	client.Sub("hello")
	done := make(chan struct{}, 1024)
	go func() {
		for i := 0; i < b.N; i++ {
			sub := q.(*queue).chanSub("hello")
			msg := NewMessageCallback(1, "", 0, nil, func(msg *Message) {
				done <- struct{}{}
			})
			sub.high <- msg
		}
	}()
	go func() {
		for i := 0; i < b.N; i++ {
			msg := <-client.Recv()
			client.Reply(msg)
		}
	}()
	for i := 0; i < b.N; i++ {
		<-done
	}
}

func BenchmarkChanSubCallback2(b *testing.B) {
	q := New("channel")
	client := q.Client()
	client.Sub("hello")
	go func() {
		for i := 0; i < b.N; i++ {
			sub := q.(*queue).chanSub("hello")
			done := make(chan struct{}, 1)
			msg := NewMessageCallback(1, "", 0, nil, func(msg *Message) {
				done <- struct{}{}
			})
			sub.high <- msg
			<-done
		}
	}()
	for i := 0; i < b.N; i++ {
		msg := <-client.Recv()
		client.Reply(msg)
	}
}

func TestChannelClose(t *testing.T) {
	//send timeout and recv timeout
	q := New("channel")

	//mempool
	done := make(chan struct{}, 1)
	go func() {
		client := q.Client()
		client.Sub("mempool")
		for {
			select {
			case msg := <-client.Recv():
				if msg == nil {
					return
				}
				if msg.Ty == types.EventTx {
					msg.Reply(client.NewMessage("mempool", types.EventReply, types.Reply{IsOk: true, Msg: []byte("word")}))
				}
			case <-done:
				client.Close()
				return
			}
		}
	}()
	client := q.Client()
	go q.Start()
	//rpc 模块 会向其他模块发送消息，自己本身不需要订阅消息
	go func() {
		done <- struct{}{}
	}()
	for i := 0; i < 10000; i++ {
		msg := client.NewMessage("mempool", types.EventTx, "hello")
		err := client.SendTimeout(msg, true, 0)
		if err == types.ErrChannelClosed {
			return
		}
		if err != nil {
			t.Error(err)
			return
		}
		_, err = client.Wait(msg)
		if err == types.ErrChannelClosed {
			return
		}
		if err != nil {
			t.Error(err)
		}
	}
}
