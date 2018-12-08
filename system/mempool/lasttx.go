package mempool

import (
	"github.com/33cn/chain33/common/listmap"
	"github.com/33cn/chain33/types"
)

//LastTxCache 最后放入cache的交易
type LastTxCache struct {
	max int
	l   *listmap.ListMap
}

//NewLastTxCache 创建最后交易的cache
func NewLastTxCache(size int) *LastTxCache {
	return &LastTxCache{
		max: size,
		l:   listmap.New(),
	}
}

//GetLatestTx 返回最新十条加入到txCache的交易
func (cache *LastTxCache) GetLatestTx() (txs []*types.Transaction) {
	cache.l.Walk(func(v interface{}) bool {
		txs = append(txs, v.(*types.Transaction))
		return true
	})
	return txs
}

//Remove remove tx of last cache
func (cache *LastTxCache) Remove(tx *types.Transaction) {
	cache.l.Remove(string(tx.Hash()))
}

//Push tx into LastTxCache
func (cache *LastTxCache) Push(tx *types.Transaction) {
	if cache.l.Size() >= cache.max {
		v := cache.l.GetTop()
		if v != nil {
			cache.Remove(v.(*types.Transaction))
		}
	}
	cache.l.Push(string(tx.Hash()), tx)
}
