package queue

import (
	"errors"
	"sync"
)

type FN_MsgCallback func(*Message) (string, int64, interface{}, error)

// QueueFuncMap 处理queue模块消息队列灰掉函数的映射管理器
type FuncMap struct {
	mtx     sync.RWMutex
	funcmap map[int]FN_MsgCallback
}

func NewFuncMap() *FuncMap {
	qfm := &FuncMap{}
	qfm.funcmap = make(map[int]FN_MsgCallback)
	return qfm
}

func (qfm *FuncMap) Register(msgid int, fn FN_MsgCallback) error {
	qfm.mtx.Lock()
	defer qfm.mtx.Unlock()
	if _, ok := qfm.funcmap[msgid]; ok {
		return errors.New("ErrMessageIDExisted")
	}
	qfm.funcmap[msgid] = fn
	return nil
}

func (qfm *FuncMap) UnRegister(msgid int) {
	qfm.mtx.Lock()
	defer qfm.mtx.Unlock()
	delete(qfm.funcmap, msgid)
}

func (qfm *FuncMap) Process(msg *Message) (bool, string, int64, interface{}, error) {
	qfm.mtx.RLock()
	defer qfm.mtx.RUnlock()
	msgid := int(msg.Ty)
	fn, ok := qfm.funcmap[msgid]
	if !ok {
		return false, "", 0, nil, nil
	}
	topic, retty, reply, err := fn(msg)
	return true, topic, retty, reply, err
}
