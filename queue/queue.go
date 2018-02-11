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

const (
	DefaultChanBuffer    = 64
	DefaultLowChanBuffer = 40960
)

func SetLogLevel(level int) {

}

func DisableLog() {
	qlog.SetHandler(log.DiscardHandler())
}

type chanSub struct {
	high    chan Message
	low     chan Message
	isClose int32
}

type Queue struct {
	chanSubs map[string]chanSub
	mu       sync.Mutex
	done     chan struct{}
	isClose  int32
}

func New(name string) *Queue {
	return &Queue{chanSubs: make(map[string]chanSub), done: make(chan struct{}, 1)}
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
	return atomic.LoadInt32(&q.isClose) == 1
}

func (q *Queue) Close() {
	if q.IsClosed() {
		return
	}
	q.mu.Lock()
	for topic, ch := range q.chanSubs {
		if ch.isClose == 0 {
			ch.high <- Message{}
			ch.low <- Message{}
			q.chanSubs[topic] = chanSub{isClose: 1}
		}
	}
	q.mu.Unlock()
	q.done <- struct{}{}
	close(q.done)
	atomic.StoreInt32(&q.isClose, 1)
	qlog.Info("queue module closed")
}

func (q *Queue) chanSub(topic string) chanSub {
	q.mu.Lock()
	defer q.mu.Unlock()
	_, ok := q.chanSubs[topic]
	if !ok {
		q.chanSubs[topic] = chanSub{make(chan Message, DefaultChanBuffer), make(chan Message, DefaultLowChanBuffer), 0}
	}
	return q.chanSubs[topic]
}

func (q *Queue) closeTopic(topic string) {
	q.mu.Lock()
	defer q.mu.Unlock()
	sub, ok := q.chanSubs[topic]
	if !ok {
		return
	}
	if sub.isClose == 0 {
		sub.high <- Message{}
		sub.low <- Message{}
	}
	q.chanSubs[topic] = chanSub{isClose: 1}
}

func (q *Queue) Send(msg Message) (err error) {
	if q.IsClosed() {
		return types.ErrChannelClosed
	}
	sub := q.chanSub(msg.Topic)
	if sub.isClose == 1 {
		return types.ErrChannelClosed
	}
	defer func() {
		res := recover()
		if res != nil {
			err = res.(error)
		}
	}()
	timeout := time.After(time.Second * 60)
	select {
	case sub.high <- msg:
	case <-timeout:
		return types.ErrTimeout
	}
	qlog.Debug("send ok", "msg", msg)
	return nil
}

func (q *Queue) SendAsyn(msg Message) error {
	if q.IsClosed() {
		return types.ErrChannelClosed
	}
	sub := q.chanSub(msg.Topic)
	if sub.isClose == 1 {
		return types.ErrChannelClosed
	}
	select {
	case sub.low <- msg:
		qlog.Debug("send asyn ok", "msg", msg)
		return nil
	default:
		qlog.Error("send asyn ok", "msg", msg)
		return types.ErrChannelFull
	}

}

func (q *Queue) NewClient() Client {
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
		qlog.Debug("reply a empty chreply", "msg", msg)
		return
	}
	msg.ChReply <- replyMsg
	qlog.Debug("reply msg ok", "msg", msg)
}

func (msg Message) String() string {
	return fmt.Sprintf("{topic:%s, Ty:%s, Id:%d, Err:%v, Ch:%v}", msg.Topic,
		types.GetEventName(int(msg.Ty)), msg.Id, msg.Err(), msg.ChReply != nil)
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
