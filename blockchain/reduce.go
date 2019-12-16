// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"container/list"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"sync"
	"time"
)

// ReduceLocalDB 实时精简localdb
func (chain *BlockChain) ReduceLocalDB() {
	defer chain.reducewg.Done()

	flagHeight, err := chain.blockStore.loadFlag(types.ReduceLocaldbHeight)
	if err != nil {
		panic(err)
	}
	if flagHeight < 0 {
		flagHeight = 0
	}
	// 1分钟检测一次是否可以进行reduce localdb
	checkTicker := time.NewTicker(10 * time.Second)
	for {
		select {
		case <-chain.quit:
			return
		case <-checkTicker.C:
			flagHeight = chain.TryReduceLocalDB(flagHeight, 100)
		}
	}
}

// TryReduce try reduce
func (chain *BlockChain) TryReduceLocalDB(flagHeight int64, rangeHeight int64) (newHeight int64) {
	if rangeHeight <= 0 {
		rangeHeight = 100
	}
	height := chain.GetBlockHeight()
	safetyHeight := height - MaxRollBlockNum
	if safetyHeight/rangeHeight > flagHeight/rangeHeight {    // 每隔rangeHeight区块进行一次精简
		chain.blockStore.reduceLocaldb(flagHeight, safetyHeight, true, chain.blockStore.reduceBody,
			func(batch dbm.Batch, height int64) {
				// 记录的时候记录下一个，中断开始执行的就是下一个
				height++
				batch.Set(types.ReduceLocaldbHeight, types.Encode(&types.Int64{Data:height}))
			})
		flagHeight = safetyHeight + 1
		chainlog.Debug("reduceLocaldb ticker", "current height", flagHeight)
		return flagHeight
	}
	return flagHeight
}

// FIFO fifo queue
type FIFO struct {
	m  map[interface{}]*list.Element
	l  *list.List
	size int
	sync.RWMutex
}

type entry struct {
	key   interface{}
	value interface{}
}

// NewFIFO  new fifi queue
func NewFIFO(size int) *FIFO {
	if size <= 0{
		size = 1
	}
	return &FIFO{
		m: make(map[interface{}]*list.Element, size),
		l: list.New(),
		size: size,
	}
}

// Contains check is a key is in the cache
func (fi *FIFO) Contains(key interface{}) bool {
	fi.RLock()
	defer fi.RUnlock()
	_, ok := fi.m[key]
	return ok
}

func (fi *FIFO) Get(key interface{}) (value interface{}, ok bool) {
	fi.Lock()
	defer fi.Unlock()

	if elem, ok := fi.m[key]; ok {
		return elem.Value.(*entry).value, true
	}
	return
}

func (fi *FIFO) Add(key, value interface{}) {
	fi.Lock()
	defer fi.Unlock()

	if fi.l.Len() >= fi.size {
		ent := fi.l.Remove(fi.l.Front()).(*entry)
		delete(fi.m, ent.key)
	}

	ent := &entry{key, value}
	elem := fi.l.PushBack(ent)
	fi.m[key] = elem
}

func (fi *FIFO) Remove(key interface{}) (present bool) {
	fi.Lock()
	defer fi.Unlock()

	if elem, ok := fi.m[key]; ok {
		ent := fi.l.Remove(elem).(*entry)
		delete(fi.m, ent.key)
		return true
	}
	return false
}
