package listmap

import (
	"container/list"

	"github.com/33cn/chain33/types"
)

type ListMap struct {
	m map[string]*list.Element
	l *list.List
}

func New() *ListMap {
	return &ListMap{
		m: make(map[string]*list.Element),
		l: list.New(),
	}
}

func (lm *ListMap) Size() int {
	return len(lm.m)
}

func (lm *ListMap) Exist(key string) bool {
	_, ok := lm.m[key]
	return ok
}

func (lm *ListMap) GetItem(key string) (interface{}, error) {
	item, ok := lm.m[key]
	if !ok {
		return nil, types.ErrNotFound
	}
	return item.Value, nil
}

func (lm *ListMap) Push(key string, value interface{}) {
	if elm, ok := lm.m[key]; ok {
		elm.Value = value
		return
	}
	elm := lm.l.PushBack(value)
	lm.m[key] = elm
	return
}

func (lm *ListMap) GetTop() interface{} {
	elm := lm.l.Front()
	if elm == nil {
		return nil
	}
	return elm.Value
}

func (lm *ListMap) Remove(key string) interface{} {
	if elm, ok := lm.m[key]; ok {
		value := lm.l.Remove(elm)
		delete(lm.m, key)
		return value
	}
	return nil
}

func (lm *ListMap) Walk(cb func(value interface{}) bool) {
	for e := lm.l.Front(); e != nil; e = e.Next() {
		if cb == nil {
			return
		}
		if !cb(e.Value) {
			return
		}
	}
}
