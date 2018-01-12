package queue

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

//消息队列的主要作用是解耦合，让各个模块相对的独立运行。
//每个模块都会有一个client 对象
//主要的操作大致如下：
// client := queue.NewClient()
// client.Sub("topicname")
// for msg := range client.Recv() {
//     process(msg)
// }

// process 函数会调用 处理具体的消息逻辑
var gId int64

type IClient interface {
	Send(msg Message, wait bool) (err error)     //异步发送消息
	SendAsyn(msg Message, wait bool) (err error) //异步发送消息
	Wait(msg Message) (Message, error)           //等待消息处理完成
	Recv() chan Message
	Sub(topic string) //订阅消息
	Close()
	NewMessage(topic string, ty int64, data interface{}) (msg Message)
}

type Client struct {
	q        *Queue
	recv     chan Message
	mu       sync.Mutex
	cache    []Message
	isclosed int32
}

func newClient(q *Queue) IClient {
	client := &Client{}
	client.q = q
	client.recv = make(chan Message, 2)
	return client
}

//1. 系统保证send出去的消息就是成功了，除非系统崩溃
//2. 系统保证每个消息都有对应的 response 消息
func (client *Client) Send(msg Message, wait bool) (err error) {
	if !wait {
		msg.ChReply = nil
		return client.q.SendAsyn(msg)
	}
	client.q.Send(msg)
	return nil
}

//系统设计出两种优先级别的消息发送
//1. SendAsyn 低优先级
//2. Send 高优先级别的发送消息
func (client *Client) SendAsyn(msg Message, wait bool) (err error) {
	if !wait {
		msg.ChReply = nil
	}
	return client.q.SendAsyn(msg)
}

func (client *Client) NewMessage(topic string, ty int64, data interface{}) (msg Message) {
	id := atomic.AddInt64(&gId, 1)
	return NewMessage(id, topic, ty, data)
}

func (client *Client) Wait(msg Message) (Message, error) {
	if msg.ChReply == nil {
		return Message{}, errors.New("empty wait channel")
	}
	timeout := time.After(time.Second * 60)
	select {
	case msg = <-msg.ChReply:
		return msg, msg.Err()
	case <-timeout:
		panic("wait for message timeout.")
	}
}

func (client *Client) Recv() chan Message {
	return client.recv
}

func (client *Client) Close() {
	atomic.StoreInt32(&client.isclosed, 1)
	close(client.recv)
}

func (client *Client) Sub(topic string) {
	highChan, lowChan := client.q.getChannel(topic)
	go func() {
		for {
			select {
			case data := <-highChan:
				if atomic.LoadInt32(&client.isclosed) == 1 {
					return
				}
				client.recv <- data
			default:
				select {
				case data := <-highChan:
					if atomic.LoadInt32(&client.isclosed) == 1 {
						return
					}
					client.recv <- data
				case data := <-lowChan:
					if atomic.LoadInt32(&client.isclosed) == 1 {
						return
					}
					client.recv <- data
				}
			}
		}
	}()
}
