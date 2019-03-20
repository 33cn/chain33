// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"fmt"
	"sync"
	"time"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	farm "github.com/dgryski/go-farm"
	"github.com/hashicorp/golang-lru"
)

// MemTreeOpera memtree操作接口
type MemTreeOpera interface {
	Add(key, value interface{})
	Get(key interface{}) (value interface{}, ok bool)
	Delete(key interface{})
	Contains(key interface{}) bool
	Len() int
}

// TreeMap map形式memtree
type TreeMap struct {
	mpCache map[interface{}]interface{}
	lock    sync.RWMutex
}

// NewTreeMap new mem tree
func NewTreeMap(size int) *TreeMap {
	mp := &TreeMap{}
	mp.mpCache = make(map[interface{}]interface{}, size)
	return mp
}

// Add 添加元素
func (tm *TreeMap) Add(key, value interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	if _, ok := tm.mpCache[key]; ok {
		delete(tm.mpCache, key)
		return
	}
	tm.mpCache[key] = value
}

// Get 获取元素
func (tm *TreeMap) Get(key interface{}) (value interface{}, ok bool) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	if value, ok := tm.mpCache[key]; ok {
		return value, ok
	}
	return nil, false
}

// Delete 删除元素
func (tm *TreeMap) Delete(key interface{}) {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	if _, ok := tm.mpCache[key]; ok {
		delete(tm.mpCache, key)
	}
}

// Contains 查看是否包含元素
func (tm *TreeMap) Contains(key interface{}) bool {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	if _, ok := tm.mpCache[key]; ok {
		return true
	}
	return false
}

// Len 元素长度
func (tm *TreeMap) Len() int {
	tm.lock.Lock()
	defer tm.lock.Unlock()
	return len(tm.mpCache)
}

// TreeARC lru的mem tree
type TreeARC struct {
	arcCache *lru.ARCCache
}

// NewTreeARC new lru mem tree
func NewTreeARC(size int) *TreeARC {
	ma := &TreeARC{}
	var err error
	ma.arcCache, err = lru.NewARC(size)
	if err != nil {
		panic("New tree lru fail")
	}
	return ma
}

// Add 添加元素
func (ta *TreeARC) Add(key, value interface{}) {
	if ta.arcCache.Contains(key) {
		ta.arcCache.Remove(key)
		return
	}
	ta.arcCache.Add(key, value)
}

// Get 获取元素
func (ta *TreeARC) Get(key interface{}) (value interface{}, ok bool) {
	return ta.arcCache.Get(key)
}

// Delete 删除元素
func (ta *TreeARC) Delete(key interface{}) {
	ta.arcCache.Remove(key)
}

// Contains 查看是否包含元素
func (ta *TreeARC) Contains(key interface{}) bool {
	return ta.arcCache.Contains(key)
}

// Len 元素长度
func (ta *TreeARC) Len() int {
	return ta.arcCache.Len()
}

// LoadTree2MemDb 从数据库中载入mem tree
func LoadTree2MemDb(db dbm.DB, hash []byte, mp map[uint64]struct{}) {
	nDb := newNodeDB(db, true)
	node, err := nDb.getLightNode(nil, hash)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	pri := ""
	if len(node.hash) > 32 {
		pri = string(node.hash[:16])
	}
	treelog.Info("hash node", "hash pri", pri, "hash", common.ToHex(node.hash), "height", node.height)
	start := time.Now()
	leftHash := make([]byte, len(node.leftHash))
	copy(leftHash, node.leftHash)
	rightHash := make([]byte, len(node.rightHash))
	copy(rightHash, node.rightHash)
	mp[farm.Hash64(node.hash)] = struct{}{}
	node.loadNodeInfo(nDb, mp)
	end := time.Now()
	treelog.Info("hash node", "cost time", end.Sub(start), "node count", len(mp))
	PrintMemStats(1)
}

func (node *Node) loadNodeInfo(db *nodeDB, mp map[uint64]struct{}) {
	if node.height == 0 {
		//trMem.Add(Hash64(node.hash), &hashNode{leftHash: node.leftHash, rightHash: node.rightHash})
		leftHash := make([]byte, len(node.leftHash))
		copy(leftHash, node.leftHash)
		rightHash := make([]byte, len(node.rightHash))
		copy(rightHash, node.rightHash)
		mp[farm.Hash64(node.hash)] = struct{}{}
		return
	}
	if node.leftHash != nil {
		left, err := db.getLightNode(nil, node.leftHash)
		if err != nil {
			return
		}
		//trMem.Add(Hash64(left.hash), &hashNode{leftHash: left.leftHash, rightHash: left.rightHash})
		leftHash := make([]byte, len(left.leftHash))
		copy(leftHash, left.leftHash)
		rightHash := make([]byte, len(left.rightHash))
		copy(rightHash, left.rightHash)
		mp[farm.Hash64(left.hash)] = struct{}{}
		left.loadNodeInfo(db, mp)
	}
	if node.rightHash != nil {
		right, err := db.getLightNode(nil, node.rightHash)
		if err != nil {
			return
		}
		//trMem.Add(Hash64(right.hash), &hashNode{leftHash: right.leftHash, rightHash: right.rightHash})
		leftHash := make([]byte, len(right.leftHash))
		copy(leftHash, right.leftHash)
		rightHash := make([]byte, len(right.rightHash))
		copy(rightHash, right.rightHash)
		mp[farm.Hash64(right.hash)] = struct{}{}
		right.loadNodeInfo(db, mp)
	}
}

func (ndb *nodeDB) getLightNode(t *Tree, hash []byte) (*Node, error) {
	// Doesn't exist, load from db.
	var buf []byte
	buf, err := ndb.db.Get(hash)

	if len(buf) == 0 || err != nil {
		return nil, ErrNodeNotExist
	}
	node, err := MakeNode(buf, t)
	if err != nil {
		panic(fmt.Sprintf("Error reading IAVLNode. bytes: %X  error: %v", buf, err))
	}
	node.hash = hash
	node.key = nil
	node.value = nil
	return node, nil
}

func copyBytes(b []byte) (copiedBytes []byte) {
	if b == nil {
		return nil
	}
	copiedBytes = make([]byte, len(b))
	copy(copiedBytes, b)
	return copiedBytes
}
