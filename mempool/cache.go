package mempool

import (
	"container/list"
	"time"

	"code.aliyun.com/chain33/chain33/account"
	"code.aliyun.com/chain33/chain33/types"
)

//--------------------------------------------------------------------------------
// Module txCache

type txCache struct {
	size       int
	txMap      map[string]*list.Element
	txList     *list.List
	txFrontTen []*types.Transaction
	accMap     map[string][]*types.Transaction
}

// Item为Mempool中包装交易的数据结构
type Item struct {
	value     *types.Transaction
	priority  int64
	enterTime int64
}

// NewTxCache初始化txCache
func newTxCache(cacheSize int64) *txCache {
	return &txCache{
		size:       int(cacheSize),
		txMap:      make(map[string]*list.Element, cacheSize),
		txList:     list.New(),
		txFrontTen: make([]*types.Transaction, 0),
		accMap:     make(map[string][]*types.Transaction),
	}
}

// txCache.LowestFee返回txHeap第一个元素的交易Fee
//func (cache *txCache) LowestFee() int64 {
//	if cache.Size() == 0 {
//		return types.MinFee
//	}
//	return cache.txLlrb.Min().(*Item).priority
//}

// txCache.TxNumOfAccount返回账户在Mempool中交易数量
func (cache *txCache) TxNumOfAccount(addr string) int64 {
	return int64(len(cache.accMap[addr]))
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

	if cache.txList.Len() >= cache.size {
		return types.ErrMemFull
	}

	it := &Item{value: tx, priority: tx.Fee, enterTime: time.Now().UnixNano() / 1000000}
	txElement := cache.txList.PushBack(it)
	cache.txMap[string(tx.Hash())] = txElement

	// 账户交易数量
	accountAddr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	cache.accMap[accountAddr] = append(cache.accMap[accountAddr], tx)

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
	cache.txList.Remove(cache.txMap[string(tx.Hash())])
	delete(cache.txMap, string(tx.Hash()))
	// 账户交易数量减1
	addr := account.PubKeyToAddress(tx.GetSignature().GetPubkey()).String()
	if cache.TxNumOfAccount(addr) > 0 {
		cache.AccountTxNumDecrease(addr, tx)
	}
}

// txCache.Size返回txCache中已存tx数目
func (cache *txCache) Size() int {
	return cache.txList.Len()
}

// txCache.SetSize用来设置Mempool容量
func (cache *txCache) SetSize(newSize int) {
	if cache.txList.Len() > 0 {
		panic("only can set a empty size")
	}
	cache.size = newSize
}

func (cache *txCache) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	var res *types.TransactionDetails
	for _, addr := range addrs.Addrs {
		if value, ok := cache.accMap[addr]; ok {
			for _, v := range value {
				txAmount, err := v.Amount()
				if err != nil {
					continue
				}
				res.Txs = append(res.Txs,
					&types.TransactionDetail{
						Tx:         v,
						Amount:     txAmount,
						Fromaddr:   addr,
						ActionName: v.ActionName(),
					})
			}
		}
	}
	return res
}

// txCache.AccountTxNumDecrease使得账户的交易计数减一
func (cache *txCache) AccountTxNumDecrease(addr string, tx *types.Transaction) {
	if value, ok := cache.accMap[addr]; ok {
		for i, t := range value {
			if string(t.Hash()) == string(tx.Hash()) {
				cache.accMap[addr] = append(cache.accMap[addr][:i], cache.accMap[addr][i+1:]...)
				return
			}
		}
	}
}
