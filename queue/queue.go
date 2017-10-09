package queue

//消息队列：
//多对多消息队列
//消息：topic

const DefaultChanBuffer = 1024

type Queue struct {
	chans map[string]chan Message
}

func New(name string) *Queue {
	chs := make(map[string]chan Message)
	return &Queue{chans: chs}
}

func (q *Queue) Start() {
	select {}
}

func (q *Queue) Send(msg Message) {
	_, ok := q.chans[msg.Topic]
	if !ok {
		q.chans[msg.Topic] = make(chan Message, DefaultChanBuffer)
	}
	q.chans[msg.Topic] <- msg
}

func (q *Queue) GetClient() IClient {
	return newClient(q)
}

type Message struct {
	Topic string
	Ty    int64
	Id    int64
	Data  interface{}
}
