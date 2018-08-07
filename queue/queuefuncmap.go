package queue

import "errors"

type msgcallback func(*Message) (string, int64, interface{}, error)

// QueueFuncMap 处理queue模块消息队列灰掉函数的映射管理器
type FuncMap struct {
	funcmap map[int]msgcallback
}

func (qfm *FuncMap) Init() {
	qfm.funcmap = make(map[int]msgcallback)
}

func (qfm *FuncMap) Register(msgid int, fn msgcallback) error {
	if _, ok := qfm.funcmap[msgid]; ok {
		return errors.New("ErrMessageIDExisted")
	}
	qfm.funcmap[msgid] = fn
	return nil
}

func (qfm *FuncMap) UnRegister(msgid int) {
	delete(qfm.funcmap, msgid)
}

func (qfm *FuncMap) Process(msg *Message) (bool, string, int64, interface{}, error) {
	msgid := int(msg.Ty)
	fn, ok := qfm.funcmap[msgid]
	if !ok {
		return false, "", 0, nil, nil
	}
	topic, retty, reply, err := fn(msg)
	return true, topic, retty, reply, err
}
