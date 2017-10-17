package queue

import (
	"sync"
)

//消息队列：
//多对多消息队列
//消息：topic

const DefaultChanBuffer = 1024

type Queue struct {
	chans  map[string]chan Message
	replys map[string]chan Message
	topic  string
	mu     sync.Mutex
}

func New(name string) *Queue {
	chs := make(map[string]chan Message)
	reps := make(map[string]chan Message)
	return &Queue{chans: chs, replys: reps, topic: name}
}

func (q *Queue) Start() {
	for {

	}
}

func (q *Queue) getChannel(topic string) (chan Message, chan Message) { //返回Queue的两个chan 队列
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.chans[topic]
	if !ok {
		q.chans[topic] = make(chan Message, DefaultChanBuffer)  //每一个topic 下面有 1024个容量
		q.replys[topic] = make(chan Message, DefaultChanBuffer) //每一个topic 下面有 1024个容量
	}
	ch1 := q.chans[topic]
	ch2 := q.replys[topic]

	return ch1, ch2
}

func (q *Queue) Send(msg Message) {

	chrecv, chreply := q.getChannel(msg.Topic)
	if msg.ParentId == 0 { //如果parentid是0，则发送到chrecv 队列，否则发送到 chreply 队列中

		chrecv <- msg
	} else {

		chreply <- msg //返回结果
	}
}

func (q *Queue) GetClient() IClient { //创建一个队列客户端
	return newClient(q)
}

type Message struct {
	Topic    string
	Ty       int64
	Id       int64
	ParentId int64
	Data     interface{}
}

func (msg *Message) GetData() interface{} { //mesage 函数， 获取消息数据
	if _, ok := msg.Data.(error); ok {
		return nil
	}
	return msg.Data
}

func (msg *Message) Err() error { //返回错误信息，如果有的话
	if err, ok := msg.Data.(error); ok {
		return err
	}
	return nil
}
