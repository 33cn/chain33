package mempool

import (
	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/common/listmap"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
)

var shashlog = log.New("module", "mempool.shash")

// SHashTxCache 通过shorthash缓存交易
type SHashTxCache struct {
	max int
	l   *listmap.ListMap
}

// NewSHashTxCache 创建通过短hash交易的cache
func NewSHashTxCache(size int) *SHashTxCache {
	return &SHashTxCache{
		max: size,
		l:   listmap.New(),
	}
}

// GetSHashTxCache 返回shorthash对应的tx交易信息
func (cache *SHashTxCache) GetSHashTxCache(sHash string) *types.Transaction {
	tx, err := cache.l.GetItem(sHash)
	if err != nil {
		return nil
	}
	return tx.(*types.Transaction)

}

// Remove remove tx of SHashTxCache
func (cache *SHashTxCache) Remove(txHash string) {
	cache.l.Remove(types.CalcTxShortHash(types.Str2Bytes(txHash)))
	//shashlog.Debug("SHashTxCache:Remove", "shash", types.CalcTxShortHash(txhash), "txhash", common.ToHex(txhash))
}

// Push tx into SHashTxCache
func (cache *SHashTxCache) Push(tx *types.Transaction, txHash []byte) {
	shash := types.CalcTxShortHash(txHash)

	if cache.Exist(shash) {
		shashlog.Error("SHashTxCache:Push:Exist", "oldhash", common.ToHex(cache.GetSHashTxCache(shash).Hash()), "newhash", common.ToHex(tx.Hash()))
		return
	}
	if cache.l.Size() >= cache.max {
		shashlog.Error("SHashTxCache:Push:ErrMemFull", "cache.l.Size()", cache.l.Size(), "cache.max", cache.max)
		return
	}
	cache.l.Push(shash, tx)
	//shashlog.Debug("SHashTxCache:Push", "shash", shash, "txhash", common.ToHex(tx.Hash()))
}

// Exist 是否存在
func (cache *SHashTxCache) Exist(shash string) bool {
	return cache.l.Exist(shash)
}
