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
	Send(msg Message) (err error)                  //异步发送消息
	Wait(msg Message)                              //等待消息处理完成
	Sub(topic string) (ch chan Message, err error) //订阅消息
}

type Message struct {
	Topic string
	Ty    int64
	Id    int64
	Data  interface{}
	Fn    func(msg Message)
}
