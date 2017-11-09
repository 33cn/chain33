package queue

import (
	"fmt"
	"sync"

	"code.aliyun.com/chain33/chain33/types"

	log "github.com/inconshreveable/log15"
)

//消息队列：
//多对多消息队列
//消息：topic

//1. 队列特点：
//1.1 一个topic 只有一个订阅者（以后会变成多个）目前基本够用，模块都只有一个实例.
//1.2 消息的回复直接通过消息自带的channel 回复
var qlog = log.New("module", "queue")

const DefaultChanBuffer = 1024

func SetLogLevel(level int) {

}

func DisableLog() {
	qlog.SetHandler(log.DiscardHandler())
}

type Queue struct {
	chans map[string]chan Message
	mu    sync.Mutex
	done  chan struct{}
}

func New(name string) *Queue {
	chs := make(map[string]chan Message)
	return &Queue{chans: chs, done: make(chan struct{})}
}

func (q *Queue) Start() {
	select {
	case <-q.done:
		break
	}
}

func (q *Queue) Close() {
	q.done <- struct{}{}
	close(q.done)
}

func (q *Queue) getChannel(topic string) chan Message {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.chans[topic]
	if !ok {
		q.chans[topic] = make(chan Message, DefaultChanBuffer)
	}
	ch1 := q.chans[topic]
	return ch1
}

func (q *Queue) Send(msg Message) {
	chrecv := q.getChannel(msg.Topic)
	chrecv <- msg
	qlog.Info("send ok", "msg", msg)
}

func (q *Queue) GetClient() IClient {
	return newClient(q)
}

type Message struct {
	Topic   string
	Ty      int64
	Id      int64
	Data    interface{}
	ChReply chan Message
}

func (msg Message) GetData() interface{} {
	if _, ok := msg.Data.(error); ok {
		return nil
	}
	return msg.Data
}

func (msg Message) Err() error {
	if err, ok := msg.Data.(error); ok {
		return err
	}
	return nil
}

func (msg Message) Reply(replyMsg Message) {
	msg.ChReply <- replyMsg
	qlog.Info("reply msg ok", "msg", msg)
}

func (msg Message) String() string {
	return fmt.Sprintf("{topic:%s, Ty:%s, Id:%d, Err:%v}", msg.Topic,
		types.GetEventName(int(msg.Ty)), msg.Id, msg.Err())
}
