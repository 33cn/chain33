package mempool

import (
	//	"container/heap"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/types"
	"github.com/petar/GoLLRB/llrb"
)

//--------------------------------------------------------------------------------
// Module txCache

type txCache struct {
	size       int
	txMap      map[string]*Item
	txLlrb     *llrb.LLRB
	txFrontTen []*types.Transaction
	accountMap map[string]int64
}

// NewTxCache初始化txCache
func newTxCache(cacheSize int64) *txCache {
	return &txCache{
		size:       int(cacheSize),
		txMap:      make(map[string]*Item, cacheSize),
		txLlrb:     llrb.New(),
		txFrontTen: make([]*types.Transaction, 0),
		accountMap: make(map[string]int64),
	}
}

// txCache.LowestFee返回txHeap第一个元素的交易Fee
func (cache *txCache) LowestFee() int64 {
	if cache.Size() == 0 {
		return types.MinFee
	}
	return cache.txLlrb.Min().(*Item).priority
}

// txCache.TxNumOfAccount返回账户在Mempool中交易数量
func (cache *txCache) TxNumOfAccount(addr string) int64 {
	return cache.accountMap[addr]
}

// txCache.Exists判断txCache中是否存在给定tx
func (cache *txCache) Exists(tx *types.Transaction) bool {
	_, exists := cache.txMap[string(tx.Hash())]
	return exists
}

// txCache.Push把给定tx添加到txCache并返回true；如果tx已经存在txCache中则返回false
func (cache *txCache) Push(tx *types.Transaction) error {
	if cache.Exists(tx) {
		return types.ErrTxExist
	}

	if cache.txLlrb.Len() >= cache.size {
		expired := 0
		for _, v := range cache.txMap {
			if time.Now().UnixNano()/1000000-v.enterTime >= mempoolExpiredInterval {
				cache.txLlrb.Delete(v)
				delete(cache.txMap, string(v.value.Hash()))
				mlog.Debug("Delete expired unpacked tx", "tx", v.value)
				expired++
			}
		}
		if tx.Fee <= cache.LowestFee() {
			mlog.Error("pushToPool", "err", types.ErrMemFull.Error())
			return types.ErrMemFull
		}
		if expired == 0 {
			poppedTx := cache.txLlrb.DeleteMin().(*Item).value
			delete(cache.txMap, string(poppedTx.Hash()))
			mlog.Debug("Delete lowest-fee unpacked tx", "tx", poppedTx)
		}
	}

	it := &Item{value: tx, priority: tx.Fee, enterTime: time.Now().UnixNano() / 1000000}
	cache.txLlrb.InsertNoReplace(it)
	cache.txMap[string(tx.Hash())] = it

	// 账户交易数量
	accountAddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if _, ok := cache.accountMap[accountAddr]; ok {
		cache.accountMap[accountAddr]++
	} else {
		cache.accountMap[accountAddr] = 1
	}

	if len(cache.txFrontTen) >= 10 {
		cache.txFrontTen = cache.txFrontTen[len(cache.txFrontTen)-9:]
	}
	cache.txFrontTen = append(cache.txFrontTen, tx)

	return nil
}

// txCache.GetLatestTx返回最新十条加入到txCache的交易
func (cache *txCache) GetLatestTx() []*types.Transaction {
	return cache.txFrontTen
}

// txCache.Remove移除txCache中给定tx
func (cache *txCache) Remove(tx *types.Transaction) {
	removed := cache.txMap[string(tx.Hash())]
	if ok {
		cache.txLlrb.Delete(removed)
		delete(cache.txMap, string(tx.Hash()))
		// 账户交易数量减1
		cache.AccountTxNumDecrease(account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String())
	}
}

// txCache.Size返回txCache中已存tx数目
func (cache *txCache) Size() int {
	return cache.txLlrb.Len()
}

// txCache.SetSize用来设置Mempool容量
func (cache *txCache) SetSize(newSize int) {
	if cache.txLlrb.Len() > 0 {
		panic("only can set a empty size")
	}
	cache.size = newSize
}

// txCache.AccountTxNumDecrease使得账户的交易计数减一
func (cache *txCache) AccountTxNumDecrease(addr string) {
	if value, ok := cache.accountMap[addr]; ok {
		if value > 1 {
			cache.accountMap[addr]--
		} else {
			delete(cache.accountMap, addr)
		}
	}
}

//--------------------------------------------------------------------------------
// Module LLRB

type Item struct {
	value     *types.Transaction
	priority  int64
	enterTime int64
}

func (i Item) Less(it llrb.Item) bool {
	return i.priority < it.(*Item).priority
}

//--------------------------------------------------------------------------------
