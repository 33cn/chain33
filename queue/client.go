package queue

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

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
	done     chan struct{}
	wg       *sync.WaitGroup
	mu       sync.Mutex
	cache    []Message
	isclosed int32
}

func newClient(q *Queue) IClient {
	client := &Client{}
	client.q = q
	client.recv = make(chan Message, 2)
	client.done = make(chan struct{}, 1)
	client.wg = &sync.WaitGroup{}
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
			time.Sleep(time.Millisecond)
			continue
		}
		break
	}
	return err
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
	client.mu.Lock()
	defer client.mu.Unlock()
	return client.recv
}

func (client *Client) Close() {
	client.done <- struct{}{}
	client.wg.Wait()
	atomic.StoreInt32(&client.isclosed, 1)
	close(client.Recv())
}

func (client *Client) isClosed(data Message, ok bool) bool {
	if !ok {
		return true
	}
	if atomic.LoadInt32(&client.isclosed) == 1 {
		return true
	}
	if data.Data == nil && data.Id == 0 && data.Ty == 0 {
		return true
	}
	return false
}

func (client *Client) Sub(topic string) {
	client.wg.Add(1)
	highChan, lowChan := client.q.getChannel(topic)
	go func() {
		defer client.wg.Done()
		for {
			select {
			case data, ok := <-highChan:
				if client.isClosed(data, ok) {
					return
				}
				client.Recv() <- data
			default:
				select {
				case data, ok := <-highChan:
					if client.isClosed(data, ok) {
						return
					}
					client.Recv() <- data
				case data, ok := <-lowChan:
					if client.isClosed(data, ok) {
						return
					}
<<<<<<< HEAD
					client.recv <- data
				case <-client.done:
					return
=======
					client.Recv() <- data
>>>>>>> e8007e766a9916e35e5bede5ec4ec4a9361c7cf3
				}
			}
		}
	}()
}
