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
	Wait(Id int64) (Message, error)          //等待消息处理完成
	Recv() chan Message
	Sub(topic string) //订阅消息
	NewMessage(topic string, ty int64, parentId int64, data interface{}) (msg Message)
}

type Client struct {
	q     *Queue
	waits map[int64]chan Message
	topic string
	id    int64
	mu    sync.Mutex
}

func newClient(q *Queue) IClient {
	client := &Client{}
	client.q = q
	client.waits = make(map[int64]chan Message)
	client.topic = ""
	return client
}

func (client *Client) setWait(id int64) {
	client.mu.Lock()
	defer client.mu.Unlock()
	client.waits[id] = make(chan Message, 1)
}

func (client *Client) delWait(id int64) {
	client.mu.Lock()
	defer client.mu.Unlock()
	delete(client.waits, id)
}

func (client *Client) getWait(id int64) (chan Message, error) {
	client.mu.Lock()
	defer client.mu.Unlock()

	ch, ok := client.waits[id]
	if ok {
		return ch, nil
	}
	return nil, errors.New("No Wait Id")
}

//1. 系统保证send出去的消息就是成功了，除非系统崩溃
//2. 系统保证每个消息都有对应的 response 消息
func (client *Client) Send(msg Message, wait bool) (err error) {
	if wait {
		client.setWait(msg.Id)
	}
	client.q.Send(msg)
	return nil
}

func (client *Client) NewMessage(topic string, ty int64, parentId int64, data interface{}) (msg Message) {
	msg.Id = atomic.AddInt64(&client.id, 1)
	msg.ParentId = parentId
	msg.Data = data
	msg.Topic = topic
	return msg
}

func (client *Client) Wait(id int64) (Message, error) {
	defer client.delWait(id)
	ch, err := client.getWait(id)
	if err != nil {
		return Message{}, err
	}
	msg := <-ch
	return msg, nil
}

func (client *Client) Recv() chan Message {
	if client.topic == "" {
		panic("client not sub anything")
	}
	chrecv, _ := client.q.getChannel(client.topic)
	return chrecv
}

func (client *Client) Sub(topic string) {
	_, reply := client.q.getChannel(topic)
	client.topic = topic
	go func() {
		for msg := range reply {
			ch, err := client.getWait(msg.ParentId)
			if err != nil {
				continue
			}
			ch <- msg
		}
	}()
}
