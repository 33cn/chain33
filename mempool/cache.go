package mempool

import (
	"container/list"

	"gitlab.33.cn/chain33/chain33/types"
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

// txCache.TxNumOfAccount返回账户在Mempool中交易数量
func (cache *txCache) TxNumOfAccount(addr string) int64 {
	return int64(len(cache.accMap[addr]))
}

// txCache.Exists判断txCache中是否存在给定tx
func (cache *txCache) Exists(hash []byte) bool {
	_, exists := cache.txMap[string(hash)]
	return exists
}

// txCache.Push把给定tx添加到txCache；如果tx已经存在txCache中或Mempool已满则返回对应error
func (cache *txCache) Push(tx *types.Transaction) error {
	hash := tx.Hash()
	if cache.Exists(hash) {
		addedItem, ok := cache.txMap[string(hash)].Value.(*Item)
		if !ok {
			mlog.Error("mempoolItemError", "item", cache.txMap[string(hash)].Value)
			return types.ErrTxExist
		}
		addedTime := addedItem.enterTime
		if types.Now().Unix()-addedTime < mempoolDupResendInterval {
			return types.ErrTxExist
		} else {
			// 超过2分钟之后的重发交易返回nil，再次发送给P2P，但是不再次加入mempool
			// 并修改其enterTime，以避免该交易一直在节点间被重发
			newEnterTime := types.Now().Unix()
			resendItem := &Item{value: tx, priority: tx.Fee, enterTime: newEnterTime}
			newItem := cache.txList.InsertAfter(resendItem, cache.txMap[string(hash)])
			cache.txList.Remove(cache.txMap[string(hash)])
			cache.txMap[string(hash)] = newItem
			// ------------------
			return nil
		}
	}

	if cache.txList.Len() >= cache.size {
		return types.ErrMemFull
	}

	it := &Item{value: tx, priority: tx.Fee, enterTime: types.Now().Unix()}
	txElement := cache.txList.PushBack(it)
	cache.txMap[string(hash)] = txElement

	// 账户交易数量
	accountAddr := tx.From()
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
func (cache *txCache) Remove(hash []byte) {
	value := cache.txList.Remove(cache.txMap[string(hash)])
	delete(cache.txMap, string(hash))
	// 账户交易数量减1
	if value == nil {
		return
	}
	tx := value.(*Item).value
	addr := tx.From()
	if cache.TxNumOfAccount(addr) > 0 {
		cache.AccountTxNumDecrease(addr, hash)
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

// txCache.GetAccTxs用来获取对应账户地址（列表）中的全部交易详细信息
func (cache *txCache) GetAccTxs(addrs *types.ReqAddrs) *types.TransactionDetails {
	res := &types.TransactionDetails{}
	for _, addr := range addrs.Addrs {
		if value, ok := cache.accMap[addr]; ok {
			for _, v := range value {
				txAmount, err := v.Amount()
				if err != nil {
					// continue
					txAmount = 0
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

// txCache.AccountTxNumDecrease根据交易哈希删除对应账户的对应交易
func (cache *txCache) AccountTxNumDecrease(addr string, hash []byte) {
	if value, ok := cache.accMap[addr]; ok {
		for i, t := range value {
			if string(t.Hash()) == string(hash) {
				cache.accMap[addr] = append(cache.accMap[addr][:i], cache.accMap[addr][i+1:]...)
				if len(cache.accMap[addr]) == 0 {
					delete(cache.accMap, addr)
				}
				return
			}
		}
	}
}
