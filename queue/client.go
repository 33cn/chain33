package queue

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
	Send(msg Message) (err error) //异步发送消息
	Wait()                        //等待消息处理完成
	Recv() chan Message
	Sub(topic string) (ch chan Message) //订阅消息
}

type Client struct {
	q     *Queue
	topic string
}

func newClient(q *Queue) IClient {
	return &Client{q, ""}
}

func (client *Client) Send(msg Message) (err error) {
	client.q.Send(msg)
	return nil
}

func (client *Client) Wait() {
}

func (client *Client) Recv() chan Message {
	if client.topic == "" {
		panic("client not sub anything")
	}
	return client.q.chans[client.topic]
}

func (client *Client) Sub(topic string) chan Message {
	ch, ok := client.q.chans[topic]
	if ok {
		return ch
	}
	client.q.chans[topic] = make(chan Message, DefaultChanBuffer)
	return client.q.chans[topic]
}
