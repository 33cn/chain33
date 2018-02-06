package queue

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"unsafe"

	"code.aliyun.com/chain33/chain33/types"
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

type Client interface {
	Send(msg Message, wait bool) (err error)     //异步发送消息
	SendAsyn(msg Message, wait bool) (err error) //异步发送消息
	Wait(msg Message) (Message, error)           //等待消息处理完成
	Recv() chan Message
	Sub(topic string) //订阅消息
	Close()
	NewMessage(topic string, ty int64, data interface{}) (msg Message)
}

type client struct {
	q        *Queue
	recv     chan Message
	done     chan struct{}
	wg       *sync.WaitGroup
	topic    string
	isClosed int32
}

func newClient(q *Queue) Client {
	client := &client{}
	client.q = q
	client.recv = make(chan Message, 5)
	client.done = make(chan struct{}, 1)
	client.wg = &sync.WaitGroup{}
	return client
}

//1. 系统保证send出去的消息就是成功了，除非系统崩溃
//2. 系统保证每个消息都有对应的 response 消息
func (client *client) Send(msg Message, wait bool) (err error) {
	if !wait {
		msg.ChReply = nil
		return client.q.SendAsyn(msg)
	}
	err = client.q.Send(msg)
	if err == types.ErrTimeout {
		panic(err)
	}
	return err
}

//系统设计出两种优先级别的消息发送
//1. SendAsyn 低优先级
//2. Send 高优先级别的发送消息
func (client *client) SendAsyn(msg Message, wait bool) (err error) {
	if !wait {
		msg.ChReply = nil
	}
	//wait for sendasyn
	i := 0
	for {
		i++
		if i%1000 == 0 {
			qlog.Error("SendAsyn retry too many times", "n", i)
		}
		err = client.q.SendAsyn(msg)
		if err != nil && err != types.ErrChannelFull {
			return err
		}
		if err == types.ErrChannelFull {
			qlog.Error("SendAsyn retry")
			time.Sleep(time.Millisecond)
			continue
		}
		break
	}
	return err
}

func (client *client) NewMessage(topic string, ty int64, data interface{}) (msg Message) {
	id := atomic.AddInt64(&gId, 1)
	return NewMessage(id, topic, ty, data)
}

func (client *client) Wait(msg Message) (Message, error) {
	if msg.ChReply == nil {
		return Message{}, errors.New("empty wait channel")
	}
	timeout := time.After(time.Second * 60)
	select {
	case msg = <-msg.ChReply:
		return msg, msg.Err()
	case <-client.done:
		return Message{}, errors.New("client is closed")
	case <-timeout:
		panic("wait for message timeout.")
	}
}

func (client *client) Recv() chan Message {
	return client.recv
}

func (client *client) getTopic() string {
	address := unsafe.Pointer(&(client.topic))
	return *(*string)(atomic.LoadPointer(&address))
}

func (client *client) setTopic(topic string) {
	address := unsafe.Pointer(&(client.topic))
	atomic.StorePointer(&address, unsafe.Pointer(&topic))
}

func (client *client) Close() {
	topic := client.getTopic()
	client.q.closeTopic(topic)
	close(client.done)
	client.wg.Wait()
	atomic.StoreInt32(&client.isClosed, 1)
	close(client.Recv())
}

func (client *client) isEnd(data Message, ok bool) bool {
	if !ok {
		return true
	}
	if atomic.LoadInt32(&client.isClosed) == 1 {
		return true
	}
	if data.Data == nil && data.Id == 0 && data.Ty == 0 {
		return true
	}
	return false
}

func (client *client) Sub(topic string) {
	client.wg.Add(1)
	client.setTopic(topic)
	sub := client.q.chanSub(topic)
	go func() {
		defer client.wg.Done()
		for {
			select {
			case data, ok := <-sub.high:
				if client.isEnd(data, ok) {
					return
				}
				client.Recv() <- data
			default:
				select {
				case data, ok := <-sub.high:
					if client.isEnd(data, ok) {
						return
					}
					client.Recv() <- data
				case data, ok := <-sub.low:
					if client.isEnd(data, ok) {
						return
					}
					client.Recv() <- data
				case <-client.done:
					return
				}
			}
		}
	}()
}
