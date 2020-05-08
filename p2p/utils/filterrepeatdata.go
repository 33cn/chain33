// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// package utils p2p utils
package utils

import (
	"sync"
	"sync/atomic"
	"time"

	log "github.com/33cn/chain33/common/log/log15"

	"github.com/33cn/chain33/types"
	lru "github.com/hashicorp/golang-lru"
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

// Filterdata filter data attribute
type Filterdata struct {
	isclose    int32
	regRData   *lru.Cache
	atomicLock sync.Mutex
}

// GetAtomicLock get lock
func (f *Filterdata) GetAtomicLock() {
	f.atomicLock.Lock()
}

// ReleaseAtomicLock release lock
func (f *Filterdata) ReleaseAtomicLock() {
	f.atomicLock.Unlock()
}

// Contains  query receive data by key
func (f *Filterdata) Contains(key string) bool {
	ok := f.regRData.Contains(key)
	return ok

}

// Remove remove receive data by key
func (f *Filterdata) Remove(key string) {
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
func (f *Filterdata) ManageRecvFilter(tickTime time.Duration) {
	ticker := time.NewTicker(tickTime)
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
			if now-regtime.(int64) < timeout {
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
func (f *Filterdata) Get(key string) (interface{}, bool) {
	val, ok := f.regRData.Get(key)
	return val, ok
}

//AddWithCheckAtomic add key if key not exist with atomic lock, return true if exist
func (f *Filterdata) AddWithCheckAtomic(key string, val interface{}) (exist bool) {

	f.GetAtomicLock()
	defer f.ReleaseAtomicLock()
	if f.Contains(key) {
		return true
	}
	f.Add(key, val)
	return false
}
