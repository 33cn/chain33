// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package utils

import (
	"sync"

	lru "github.com/hashicorp/golang-lru"
)

// lru缓存封装, 控制占用空间大小
type SpaceLimitCache struct {
	maxSize  int64
	currSize int64
	sizeMap  map[interface{}]int64
	data     *lru.Cache
	lock     *sync.RWMutex
}

// NewSpaceLimitCache new space limit cache
func NewSpaceLimitCache(num int, maxByteSize int64) *SpaceLimitCache {

	cache := &SpaceLimitCache{maxSize: maxByteSize}
	cache.sizeMap = make(map[interface{}]int64)
	cache.lock = &sync.RWMutex{}
	var err error
	cache.data, err = lru.New(num)
	if err != nil {
		panic(err)
	}
	return cache
}

// Add add key val
func (c *SpaceLimitCache) Add(key interface{}, val interface{}, size int64) bool {

	c.lock.Lock()
	defer c.lock.Unlock()

	//如果存在先删除
	if c.data.Contains(key) {
		c.data.Remove(key)
		c.currSize -= c.sizeMap[key]
		delete(c.sizeMap, key)
	}

	//单个值超过最大大小
	if size > c.maxSize {
		return false
	}
	c.currSize += size
	keys := c.data.Keys()

	//超过最大大小, 移除最早的值
	for i := 0; i < len(keys) && c.currSize > c.maxSize; i++ {
		c.currSize -= c.sizeMap[keys[i]]
		c.data.RemoveOldest()
		delete(c.sizeMap, keys[i])
	}

	evicted := c.data.Add(key, val)
	c.sizeMap[key] = size
	//触发最早数据被移除, 更新目前大小
	if evicted {
		c.currSize -= c.sizeMap[keys[0]]
		delete(c.sizeMap, keys[0])
	}
	return true
}

// Get get key
func (c *SpaceLimitCache) Get(key interface{}) interface{} {
	v, _ := c.data.Get(key)
	return v
}

// Remove remove key
func (c *SpaceLimitCache) Remove(key interface{}) (interface{}, bool) {

	c.lock.Lock()
	defer c.lock.Unlock()
	val, exist := c.data.Get(key)
	if exist {
		c.data.Remove(key)
		c.currSize -= c.sizeMap[key]
		delete(c.sizeMap, key)
	}
	return val, exist
}

// Contains check if exist
func (c *SpaceLimitCache) Contains(key interface{}) bool {

	return c.data.Contains(key)
}
