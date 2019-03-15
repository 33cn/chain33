// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package queue

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"unsafe"

	"github.com/33cn/chain33/types"
)

//消息队列的主要作用是解耦合，让各个模块相对的独立运行。
//每个模块都会有一个client 对象
//主要的操作大致如下：
// client := queue.Client()
// client.Sub("topicname")
// for msg := range client.Recv() {
//     process(msg)
// }
// process 函数会调用 处理具体的消息逻辑

var gid int64

// Client 消息队列的接口，每个模块都需要一个发送接受client
type Client interface {
	Send(msg *Message, waitReply bool) (err error) //同步发送消息
	SendTimeout(msg *Message, waitReply bool, timeout time.Duration) (err error)
	Wait(msg *Message) (*Message, error)                               //等待消息处理完成
	WaitTimeout(msg *Message, timeout time.Duration) (*Message, error) //等待消息处理完成
	Recv() chan *Message
	Reply(msg *Message)
	Sub(topic string) //订阅消息
	Close()
	CloseQueue() (*types.Reply, error)
	NewMessage(topic string, ty int64, data interface{}) (msg *Message)
}

// Module be used for module interface
type Module interface {
	SetQueueClient(client Client)
	//wait for ready
	Wait()
	Close()
}

type client struct {
	q          *queue
	recv       chan *Message
	done       chan struct{}
	wg         *sync.WaitGroup
	topic      unsafe.Pointer
	isClosed   int32
	isCloseing int32
}

func newClient(q *queue) Client {
	client := &client{}
	client.q = q
	client.recv = make(chan *Message, 5)
	client.done = make(chan struct{}, 1)
	client.wg = &sync.WaitGroup{}
	return client
}

// Send 发送消息,msg 消息 ,waitReply 是否等待回应
//1. 系统保证send出去的消息就是成功了，除非系统崩溃
//2. 系统保证每个消息都有对应的 response 消息
func (client *client) Send(msg *Message, waitReply bool) (err error) {
	timeout := time.Duration(-1)
	err = client.SendTimeout(msg, waitReply, timeout)
	if err == ErrQueueTimeout {
		panic(err)
	}
	return err
}

// SendTimeout 超时发送， msg 消息 ,waitReply 是否等待回应， timeout 超时时间
func (client *client) SendTimeout(msg *Message, waitReply bool, timeout time.Duration) (err error) {
	if client.isClose() {
		return ErrIsQueueClosed
	}
	if !waitReply {
		msg.chReply = nil
		return client.q.sendLowTimeout(msg, timeout)
	}
	return client.q.send(msg, timeout)
}

//系统设计出两种优先级别的消息发送
//1. SendAsyn 低优先级
//2. Send 高优先级别的发送消息

// NewMessage 新建消息 topic模块名称 ty消息类型 data 数据
func (client *client) NewMessage(topic string, ty int64, data interface{}) (msg *Message) {
	id := atomic.AddInt64(&gid, 1)
	return NewMessage(id, topic, ty, data)
}

func (client *client) Reply(msg *Message) {
	if msg.chReply != nil {
		msg.Reply(msg)
		return
	}
	if msg.callback != nil {
		client.q.callback <- msg
	}
}

// WaitTimeout 等待时间 msg 消息 timeout 超时时间
func (client *client) WaitTimeout(msg *Message, timeout time.Duration) (*Message, error) {
	if msg.chReply == nil {
		return &Message{}, errors.New("empty wait channel")
	}
	if timeout == -1 {
		msg = <-msg.chReply
		return msg, msg.Err()
	}
	t := time.NewTimer(timeout)
	defer t.Stop()
	select {
	case msg = <-msg.chReply:
		return msg, msg.Err()
	case <-client.done:
		return &Message{}, ErrIsQueueClosed
	case <-t.C:
		return &Message{}, ErrQueueTimeout
	}
}

// Wait 等待时间
func (client *client) Wait(msg *Message) (*Message, error) {
	timeout := time.Duration(-1)
	msg, err := client.WaitTimeout(msg, timeout)
	if err == ErrQueueTimeout {
		panic(err)
	}
	return msg, err
}

// Recv 获取接受消息通道
func (client *client) Recv() chan *Message {
	return client.recv
}

func (client *client) getTopic() string {
	return *(*string)(atomic.LoadPointer(&client.topic))
}

func (client *client) setTopic(topic string) {
	// #nosec
	atomic.StorePointer(&client.topic, unsafe.Pointer(&topic))
}

func (client *client) isClose() bool {
	return atomic.LoadInt32(&client.isClosed) == 1
}

func (client *client) isInClose() bool {
	return atomic.LoadInt32(&client.isCloseing) == 1
}

// Close 关闭client
func (client *client) Close() {
	if atomic.LoadInt32(&client.isClosed) == 1 || client.topic == nil {
		return
	}
	topic := client.getTopic()
	client.q.closeTopic(topic)
	close(client.done)
	atomic.StoreInt32(&client.isCloseing, 1)
	client.wg.Wait()
	atomic.StoreInt32(&client.isClosed, 1)
	close(client.Recv())
	for msg := range client.Recv() {
		msg.Reply(client.NewMessage(msg.Topic, msg.Ty, types.ErrChannelClosed))
	}
}

// CloseQueue 关闭消息队列
func (client *client) CloseQueue() (*types.Reply, error) {
	//	client.q.Close()
	if client.q.isClosed() {
		return &types.Reply{IsOk: true}, nil
	}
	qlog.Debug("queue", "msg", "closing chain33")
	client.q.interrupt <- struct{}{}
	//	close(client.q.interupt)
	return &types.Reply{IsOk: true}, nil
}

func (client *client) isEnd(data *Message, ok bool) bool {
	if !ok {
		return true
	}
	if data.Data == nil && data.ID == 0 && data.Ty == 0 {
		return true
	}
	if atomic.LoadInt32(&client.isClosed) == 1 {
		return true
	}
	return false
}

// Sub 订阅消息类型
func (client *client) Sub(topic string) {
	//正在关闭或者已经关闭
	if client.isInClose() || client.isClose() {
		return
	}
	client.wg.Add(1)
	client.setTopic(topic)
	sub := client.q.chanSub(topic)
	go func() {
		defer func() {
			client.wg.Done()
		}()
		for {
			select {
			case data, ok := <-sub.high:
				if client.isEnd(data, ok) {
					qlog.Info("unsub1", "topic", topic)
					return
				}
				client.Recv() <- data
			default:
				select {
				case data, ok := <-sub.high:
					if client.isEnd(data, ok) {
						qlog.Info("unsub2", "topic", topic)
						return
					}
					client.Recv() <- data
				case data, ok := <-sub.low:
					if client.isEnd(data, ok) {
						qlog.Info("unsub3", "topic", topic)
						return
					}
					client.Recv() <- data
				case <-client.done:
					qlog.Error("unsub4", "topic", topic)
					return
				}
			}
		}
	}()
}
