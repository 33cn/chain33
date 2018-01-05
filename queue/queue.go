package queue

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"time"

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

const DefaultChanBuffer = 64

func SetLogLevel(level int) {

}

func DisableLog() {
	qlog.SetHandler(log.DiscardHandler())
}

type Queue struct {
	chans   map[string]chan Message
	mu      sync.Mutex
	done    chan struct{}
	isclose bool
}

func New(name string) *Queue {
	chs := make(map[string]chan Message)
	return &Queue{chans: chs, done: make(chan struct{}, 1)}
}

func (q *Queue) Start() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	// Block until a signal is received.
	select {
	case <-q.done:
		break
	case s := <-c:
		fmt.Println("Got signal:", s)
		break
	}
}

func (q *Queue) IsClosed() bool {
	q.mu.Lock()
	c := q.isclose
	q.mu.Unlock()
	return c
}

func (q *Queue) Close() {
	for _, ch := range q.chans {
		close(ch)
	}
	q.done <- struct{}{}
	close(q.done)
	q.mu.Lock()
	q.isclose = true
	q.mu.Unlock()
	qlog.Info("queue module closed")
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
	if q.IsClosed() {
		return
	}
	chrecv := q.getChannel(msg.Topic)
	timeout := time.After(time.Second * 5)
	select {
	case chrecv <- msg:
	case <-timeout:
		panic("send message timeout.")
	}
	qlog.Debug("send ok", "msg", msg)
}

func (q *Queue) SendAsyn(msg Message) error {
	if q.IsClosed() {
		return types.ErrChannelClosed
	}
	chrecv := q.getChannel(msg.Topic)
	select {
	case chrecv <- msg:
		return nil
	default:
		return types.ErrChannelFull
	}
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

func NewMessage(id int64, topic string, ty int64, data interface{}) (msg Message) {
	msg.Id = id
	msg.Ty = ty
	msg.Data = data
	msg.Topic = topic
	msg.ChReply = make(chan Message, 1)
	return msg
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
	if msg.ChReply == nil {
		qlog.Info("reply a empty chreply", "msg", msg)
		return
	}
	msg.ChReply <- replyMsg
	qlog.Debug("reply msg ok", "msg", msg)
}

func (msg Message) String() string {
	return fmt.Sprintf("{topic:%s, Ty:%s, Id:%d, Err:%v, Ch:%v}", msg.Topic,
		types.GetEventName(int(msg.Ty)), msg.Id, msg.Err(), msg.ChReply)
}

func (msg Message) ReplyErr(title string, err error) {
	var reply types.Reply
	if err != nil {
		qlog.Error(title, "err", err.Error())
		reply.IsOk = false
		reply.Msg = []byte(err.Error())
	} else {
		qlog.Debug(title, "success", "ok")
		reply.IsOk = true
	}
	id := atomic.AddInt64(&gId, 1)
	msg.Reply(NewMessage(id, "", types.EventReply, &reply))
}
