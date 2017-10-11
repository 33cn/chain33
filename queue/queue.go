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
	mu     sync.Mutex
}

func New(name string) *Queue {
	chs := make(map[string]chan Message)
	reps := make(map[string]chan Message)
	return &Queue{chans: chs, replys: reps}
}

func (q *Queue) Start() {
	select {}
}

func (q *Queue) getChannel(topic string) (chan Message, chan Message) {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.chans[topic]
	if !ok {
		q.chans[topic] = make(chan Message, DefaultChanBuffer)
		q.replys[topic] = make(chan Message, DefaultChanBuffer)
	}
	ch1 := q.chans[topic]
	ch2 := q.replys[topic]

	return ch1, ch2
}

func (q *Queue) Send(msg Message) {
	chrecv, chreply := q.getChannel(msg.Topic)
	if msg.ParentId == 0 {
		chrecv <- msg
	} else {
		chreply <- msg
	}
}

func (q *Queue) GetClient() IClient {
	return newClient(q)
}

type Message struct {
	Topic    string
	Ty       int64
	Id       int64
	ParentId int64
	Data     interface{}
}

func (msg *Message) GetData() interface{} {
	if _, ok := msg.Data.(error); ok {
		return nil
	}
	return msg.Data
}

func (msg *Message) Err() error {
	if err, ok := msg.Data.(error); ok {
		return err
	}
	return nil
}
