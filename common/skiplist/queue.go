package skiplist

import (
	"container/list"

	"github.com/33cn/chain33/types"
)

//Scorer 接口实现 Value的 Score 功能
type Scorer interface {
	GetScore() int64
	Hash() []byte
	//在score相同情况下的比较
	Compare(Scorer) int
}

// Queue 价格队列模式(价格=手续费/交易字节数,价格高者优先,同价则时间早优先)
type Queue struct {
	txMap   map[string]*list.Element
	txList  *SkipList
	maxsize int64
}

// NewQueue 创建队列
func NewQueue(maxsize int64) *Queue {
	return &Queue{
		txMap:   make(map[string]*list.Element),
		txList:  NewSkipList(&SkipValue{Score: -1, Value: nil}),
		maxsize: maxsize,
	}
}

/*
为了处理相同 Score 的问题，需要一个队列保存相同 Score 下面的交易
*/
func (cache *Queue) insertSkipValue(item Scorer) *list.Element {
	skvalue := cache.CreateSkipValue(item)
	value := cache.txList.Find(skvalue)
	var txlist *list.List
	if value == nil {
		txlist = list.New()
		skvalue.Value = txlist
		cache.txList.Insert(skvalue)
	} else {
		txlist = value.Value.(*list.List)
	}
	return txlist.PushBack(item)
}

func (cache *Queue) deleteSkipValue(item *list.Element) error {
	if item == nil {
		return nil
	}
	skvalue := cache.CreateSkipValue(item.Value.(Scorer))
	value := cache.txList.Find(skvalue)
	var txlist *list.List
	if value == nil {
		return types.ErrNotFound
	}
	txlist = value.Value.(*list.List)
	txlist.Remove(item)
	if txlist.Len() == 0 {
		cache.txList.Delete(value)
	}
	return nil
}

//CreateSkipValue 创建一个 仅仅有 score 的Value
func (cache *Queue) CreateSkipValue(item Scorer) *SkipValue {
	skvalue := &SkipValue{Score: item.GetScore()}
	return skvalue
}

//Exist 是否存在
func (cache *Queue) Exist(hash string) bool {
	_, exists := cache.txMap[hash]
	return exists
}

//GetItem 获取数据通过 key
func (cache *Queue) GetItem(hash string) (Scorer, error) {
	if k, exist := cache.txMap[hash]; exist {
		return k.Value.(Scorer), nil
	}
	return nil, types.ErrNotFound
}

//Insert Scorer item to queue
func (cache *Queue) Insert(hash string, item Scorer) error {
	cache.txMap[string(hash)] = cache.insertSkipValue(item)
	return nil
}

// Push 把给定tx添加到Queue；如果tx已经存在Queue中或Mempool已满则返回对应error
func (cache *Queue) Push(item Scorer) error {
	hash := item.Hash()
	if cache.Exist(string(hash)) {
		return types.ErrTxExist
	}
	sv := cache.CreateSkipValue(item)
	if int64(cache.Size()) >= cache.maxsize {
		tail := cache.Last()
		lasthash := string(tail.Hash())
		//价格高存留
		switch sv.Compare(cache.CreateSkipValue(tail)) {
		case Big:
			cache.Remove(lasthash)
		case Equal:
			//再score 相同的情况下，item 之间的比较方法
			//权重大的留下来
			if item.Compare(tail) == Big {
				cache.Remove(lasthash)
				break
			}
			return types.ErrMemFull
		case Small:
			return types.ErrMemFull
		default:
			return types.ErrMemFull
		}
	}
	cache.Insert(string(hash), item)
	return nil
}

// Remove 删除数据
func (cache *Queue) Remove(hash string) error {
	elm, ok := cache.txMap[hash]
	if !ok {
		return types.ErrNotFound
	}
	err := cache.deleteSkipValue(elm)
	if err != nil {
		return err
	}
	delete(cache.txMap, hash)
	return nil
}

// Size 数据总数
func (cache *Queue) Size() int {
	return len(cache.txMap)
}

//Last 取出最后一个交易
func (cache *Queue) Last() Scorer {
	if cache.Size() == 0 {
		return nil
	}
	tailqueue := cache.txList.GetIterator().Last()
	tail := tailqueue.Value.(*list.List).Back().Value.(Scorer)
	return tail
}

//First 取出第一个交易
func (cache *Queue) First() Scorer {
	if cache.Size() == 0 {
		return nil
	}
	tailqueue := cache.txList.GetIterator().First()
	tail := tailqueue.Value.(*list.List).Front().Value.(Scorer)
	return tail
}

// Walk 遍历整个队列
func (cache *Queue) Walk(count int, cb func(value Scorer) bool) {
	i := 0
	cache.txList.Walk(func(item interface{}) bool {
		l := item.(*list.List)
		for e := l.Front(); e != nil; e = e.Next() {
			if !cb(e.Value.(Scorer)) {
				return false
			}
			i++
			if i == count {
				return false
			}
		}
		return true
	})
}
