package para

import (
	"container/list"
	"sync"

	"gitlab.33.cn/chain33/chain33/types"
)

//--------------------------------------------------------------------------------
// ParaChain txCache

const (
	AddAct int64 = 1
	DelAct int64 = 2 //reference blockstore.go
)

type txCache struct {
	size   int
	txMap  map[string]*list.Element
	txList *list.List
	lock   sync.RWMutex
}

// Item为Mempool中包装交易的数据结构
type Item struct {
	value *types.Transaction
	ty    int64 //AddAct or DelAct
}

// NewTxCache初始化txCache
func newTxCache(cacheSize int64) *txCache {
	return &txCache{
		size:   int(cacheSize),
		txMap:  make(map[string]*list.Element, cacheSize),
		txList: list.New(),
	}
}

// txCache.Exists判断txCache中是否存在给定tx
func (cache *txCache) Exists(hash []byte) bool {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	_, exists := cache.txMap[string(hash)]
	return exists
}

// txCache.Get获取指定hash值的tx
func (cache *txCache) Get(hash []byte) *Item {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	if v, ok := cache.txMap[string(hash)]; ok {
		return v.Value.(*Item)
	}
	return nil
}

// txCache.Push把给定tx添加到txCache；如果tx已经存在txCache中或Mempool已满则返回对应error
func (cache *txCache) Push(tx *types.Transaction, ty int64) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	hash := tx.Hash()
	if cache.txList.Len() >= cache.size {
		return types.ErrMemFull
	}

	it := &Item{value: tx, ty: ty}
	txElement := cache.txList.PushBack(it)
	cache.txMap[string(hash)] = txElement
	return nil
}

// txCache.Pull获取交易
func (cache *txCache) Pull(size int, txHashList [][]byte) []*types.Transaction {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	dupMap := make(map[string]bool)
	for i := 0; i < len(txHashList); i++ {
		dupMap[string(txHashList[i])] = true
	}

	var result []*types.Transaction
	index := 0
	txlist := cache.txList
	for v := txlist.Front(); v != nil; v = v.Next() {
		tx := v.Value.(*Item).value
		if _, ok := dupMap[string(tx.Hash())]; ok {
			continue
		}
		result = append(result, tx)
		index++
		if index == int(size) {
			break
		}
	}
	return result
}

// txCache.Remove移除txCache中给定tx
func (cache *txCache) Remove(hash []byte) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	cache.txList.Remove(cache.txMap[string(hash)])
	delete(cache.txMap, string(hash))
}

// txCache.Size返回txCache中已存tx数目
func (cache *txCache) Size() int {
	cache.lock.RLock()
	defer cache.lock.RUnlock()
	return cache.txList.Len()
}

// txCache.SetSize用来设置Mempool容量
func (cache *txCache) SetSize(newSize int) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if cache.txList.Len() > 0 {
		panic("only can set a empty size")
	}
	cache.size = newSize
}
