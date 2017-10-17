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
	Recv() chan Message                      //返回 chan Message 消息，即返回一个channal
	Sub(topic string)                        //订阅消息
	NewMessage(topic string, ty int64, parentId int64, data interface{}) (msg Message)
}

type Client struct {
	q     *Queue
	waits map[int64]chan Message //等待队列
	topic string
	id    int64
	mu    sync.Mutex
}

func newClient(q *Queue) IClient { //创建一个新的客户端
	client := &Client{}
	client.q = q
	client.waits = make(map[int64]chan Message) //创建一个map ，值是chan
	client.topic = q.topic                      //
	return client
}

func (client *Client) setWait(id int64) { //设置等待ID事件
	client.mu.Lock()
	defer client.mu.Unlock()
	client.waits[id] = make(chan Message, 1) //有缓冲的 chan ,缓冲只有 1
}

func (client *Client) delWait(id int64) { //删除等待事件ID
	client.mu.Lock()
	defer client.mu.Unlock()
	delete(client.waits, id)
}

func (client *Client) getWait(id int64) (chan Message, error) { //获取对应wait ID的channal
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
		client.setWait(msg.Id) //设置wait id
	}
	client.q.Send(msg) //调用QUEUE发送函数
	return nil
}

func (client *Client) NewMessage(topic string, ty int64, parentId int64, data interface{}) (msg Message) {
	msg.Id = atomic.AddInt64(&client.id, 1) //每次newMessage 的时候 client id 会自动累加1
	msg.ParentId = parentId
	msg.Data = data
	msg.Topic = topic
	msg.Ty = ty
	return msg
}

func (client *Client) Wait(id int64) (Message, error) { //等待一个ID事件的返回结果
	defer client.delWait(id)
	ch, err := client.getWait(id)
	if err != nil {
		return Message{}, err
	}
	msg := <-ch
	return msg, nil
}

func (client *Client) Recv() chan Message { //获取队列接受到的消息channal
	if client.topic == "" {
		panic("client not sub anything")
	}

	chrecv, _ := client.q.getChannel(client.topic)

	return chrecv
}

func (client *Client) Sub(topic string) { //订阅接收返回的消息,只要订阅了对应的消息，只需要关注Wait 函数的结果即可
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
