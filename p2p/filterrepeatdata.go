// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package p2p

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/33cn/chain33/types"
	lru "github.com/hashicorp/golang-lru"
)

// Filter  a Filter object
var (
	peerAddrFilter = NewFilter(PeerAddrCacheNum)
	//接收交易和区块过滤缓存, 避免重复提交到mempool或blockchain
	txHashFilter    = NewFilter(TxRecvFilterCacheNum)
	blockHashFilter = NewFilter(BlockFilterCacheNum)

	//发送交易和区块时过滤缓存, 解决冗余广播发送
	txSendFilter    = NewFilter(TxSendFilterCacheNum)
	blockSendFilter = NewFilter(BlockFilterCacheNum)
)

// NewFilter produce a filter object
func NewFilter(num int) *Filterdata {
	filter := new(Filterdata)
	var err error
	filter.regRData, err = lru.New(num)
	if err != nil {
		panic(err)
	}
	return filter
}

type sendFilterInfo struct {
	//记录广播交易或区块时需要忽略的节点, 这些节点可能是交易的来源节点,也可能节点间维护了多条连接, 冗余发送
	ignoreSendPeers map[string]bool
}

// Filterdata filter data attribute
type Filterdata struct {
	isclose    int32
	regRData   *lru.Cache
	atomicLock sync.Mutex
}

// GetLock get lock
func (f *Filterdata) GetLock() {
	f.atomicLock.Lock()
}

// ReleaseLock release lock
func (f *Filterdata) ReleaseLock() {
	f.atomicLock.Unlock()
}

// RegRecvData add receive data by key
func (f *Filterdata) RegRecvData(key string) bool {
	f.regRData.Add(key, time.Duration(types.Now().Unix()))
	return true
}

// QueryRecvData  query receive data by key
func (f *Filterdata) QueryRecvData(key string) bool {
	ok := f.regRData.Contains(key)
	return ok

}

// RemoveRecvData remove receive data by key
func (f *Filterdata) RemoveRecvData(key string) {
	f.regRData.Remove(key)
}

// Close the filter object
func (f *Filterdata) Close() {
	atomic.StoreInt32(&f.isclose, 1)
}

func (f *Filterdata) isClose() bool {
	return atomic.LoadInt32(&f.isclose) == 1
}

// ManageRecvFilter manager receive filter
func (f *Filterdata) ManageRecvFilter() {
	ticker := time.NewTicker(time.Second * 30)
	var timeout int64 = 60
	defer ticker.Stop()
	for {
		<-ticker.C
		now := types.Now().Unix()
		for _, key := range f.regRData.Keys() {
			regtime, exist := f.regRData.Get(key)
			if !exist {
				log.Warn("Not found in regRData", "Key", key)
				continue
			}
			if now-int64(regtime.(time.Duration)) < timeout {
				break
			}
			f.regRData.Remove(key)
		}

		if f.isClose() {
			return
		}
	}
}

// Add add val
func (f *Filterdata) Add(key string, val interface{}) bool {

	return f.regRData.Add(key, val)
}

// Get get val
func (f *Filterdata) Get(key string) interface{} {
	val, _ := f.regRData.Get(key)
	return val
}
