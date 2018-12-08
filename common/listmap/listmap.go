package listmap

import (
	"container/list"

	"github.com/33cn/chain33/types"
)

//ListMap list 和 map 组合的一个数据结构体
type ListMap struct {
	m map[string]*list.Element
	l *list.List
}

//New 创建一个新的数据结构
func New() *ListMap {
	return &ListMap{
		m: make(map[string]*list.Element),
		l: list.New(),
	}
}

//Size 结构中的item 数目
func (lm *ListMap) Size() int {
	return len(lm.m)
}

//Exist 是否存在这个元素
func (lm *ListMap) Exist(key string) bool {
	_, ok := lm.m[key]
	return ok
}

//GetItem 通过key 获取这个 item
func (lm *ListMap) GetItem(key string) (interface{}, error) {
	item, ok := lm.m[key]
	if !ok {
		return nil, types.ErrNotFound
	}
	return item.Value, nil
}

//Push 在队伍尾部插入
func (lm *ListMap) Push(key string, value interface{}) {
	if elm, ok := lm.m[key]; ok {
		elm.Value = value
		return
	}
	elm := lm.l.PushBack(value)
	lm.m[key] = elm
}

//GetTop 获取队列头部的数据
func (lm *ListMap) GetTop() interface{} {
	elm := lm.l.Front()
	if elm == nil {
		return nil
	}
	return elm.Value
}

//Remove 删除某个key
func (lm *ListMap) Remove(key string) interface{} {
	if elm, ok := lm.m[key]; ok {
		value := lm.l.Remove(elm)
		delete(lm.m, key)
		return value
	}
	return nil
}

//Walk 遍历整个结构，如果cb 返回false 那么停止遍历
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
