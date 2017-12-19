package queue

import (
	"errors"
	"sync"
	"sync/atomic"
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

type IClient interface {
	Send(msg Message, wait bool) (err error) //异步发送消息
	Wait(msg Message) (Message, error)       //等待消息处理完成
	Recv() chan Message
	Sub(topic string) //订阅消息
	Close()
	NewMessage(topic string, ty int64, data interface{}) (msg Message)
}

type Client struct {
	q        *Queue
	recv     chan Message
	id       int64
	mu       sync.Mutex
	isclosed int32
}

func newClient(q *Queue) IClient {
	client := &Client{}
	client.q = q
	client.recv = make(chan Message, DefaultChanBuffer)
	return client
}

//1. 系统保证send出去的消息就是成功了，除非系统崩溃
//2. 系统保证每个消息都有对应的 response 消息
func (client *Client) Send(msg Message, wait bool) (err error) {
	if !wait {
		msg.ChReply = nil
	}
	client.q.Send(msg)
	return nil
}

func (client *Client) NewMessage(topic string, ty int64, data interface{}) (msg Message) {
	msg.Id = atomic.AddInt64(&client.id, 1)
	msg.Ty = ty
	msg.Data = data
	msg.Topic = topic
	msg.ChReply = make(chan Message, 1)
	return msg
}

func (client *Client) Wait(msg Message) (Message, error) {
	if msg.ChReply == nil {
		return Message{}, errors.New("empty wait channel")
	}
	msg = <-msg.ChReply
	return msg, msg.Err()
}

func (client *Client) Recv() chan Message {
	return client.recv
}

func (client *Client) Close() {
	atomic.StoreInt32(&client.isclosed, 1)
	close(client.recv)
}

func (client *Client) Sub(topic string) {
	recv := client.q.getChannel(topic)
	go func() {
		for msg := range recv {
			if atomic.LoadInt32(&client.isclosed) == 0 {
				client.recv <- msg
			}
		}
	}()
}
