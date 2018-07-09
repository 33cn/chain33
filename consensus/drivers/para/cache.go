package para

import (
	"container/list"
	"sync"

	"gitlab.33.cn/chain33/chain33/types"
)

//--------------------------------------------------------------------------------
// ParaChain txCache

type txCache struct {
	size   int
	txMap  map[string]*list.Element
	txList *list.List
	lock   sync.RWMutex
}

// Item为Mempool中包装交易的数据结构
type Item struct {
	value *types.Transaction
	seq   int64
}

// NewTxCache初始化txCache
//func newTxCache(cacheSize int64) *txCache {
//	return &txCache{
//		size:   int(cacheSize),
//		txMap:  make(map[string]*list.Element, cacheSize),
//		txList: list.New(),
//	}
//}

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
func (cache *txCache) Push(tx *types.Transaction, seq int64) error {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	hash := tx.Hash()
	if cache.txList.Len() >= cache.size {
		return types.ErrMemFull
	}

	it := &Item{value: tx, seq: seq}
	txElement := cache.txList.PushBack(it)
	cache.txMap[string(hash)] = txElement
	return nil
}

// txCache.Pull获取交易
// 固定取得seqRange个数的seq中的交易，如果没有足够的seq，返回空切片
// 如果因为个数超过范围，则少取一个seq的数据
func (cache *txCache) Pull(size int, seqRange int64, txHashList [][]byte) []*types.Transaction {
	cache.lock.RLock()
	defer cache.lock.RUnlock()

	var seqTotal int64 = 0
	seqRecord := make(map[int64]int)
	dupMap := make(map[string]bool)
	for i := 0; i < len(txHashList); i++ {
		dupMap[string(txHashList[i])] = true
	}

	var result []*types.Transaction
	var tempRes []*types.Transaction

	index := 0
	txlist := cache.txList
	for v := txlist.Front(); v != nil; v = v.Next() {
		tx := v.Value.(*Item).value
		seq := v.Value.(*Item).seq
		if _, exist := seqRecord[seq]; !exist {
			seqTotal += 1
			seqRecord[seq] = 1
		} else {
			seqRecord[seq] += 1
		}

		if seqTotal > seqRange {
			break
		}

		if _, ok := dupMap[string(tx.Hash())]; ok {
			continue
		}
		result = append(result, tx)
		index++
		if index == size {
			result = result[0 : len(result)-seqRecord[seq]]
			break
		}
	}

	if seqTotal < seqRange {
		return tempRes
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

// txCache.SetSize用来设置容量
func (cache *txCache) SetSize(newSize int) {
	cache.lock.Lock()
	defer cache.lock.Unlock()
	if cache.txList.Len() > 0 {
		panic("only can set a empty size")
	}
	cache.size = newSize
}
