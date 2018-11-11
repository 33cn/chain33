// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"bytes"
	"container/list"
	"sync"

	"github.com/33cn/chain33/common"
)

const blockNodeCacheLimit = 10240 //目前best主链保存最新的10240个blocknode

// chainView provides a flat view of a specific branch of the block chain from
// its tip back to the genesis block and provides various convenience functions
// for comparing chains.
//
// For example, assume a block chain with a side chain as depicted below:
//   genesis -> 1 -> 2 -> 3 -> 4  -> 5 ->  6  -> 7  -> 8
//                         \-> 4a -> 5a -> 6a
//
// The chain view for the branch ending in 6a consists of:
//   genesis -> 1 -> 2 -> 3 -> 4a -> 5a -> 6a
type chainView struct {
	mtx        sync.Mutex
	nodes      map[int64]*list.Element
	cacheQueue *list.List
}

//需要从数据库中获取lastblock然后组装成node节点开始创建ChainView
func newChainView(tip *blockNode) *chainView {

	chainview := &chainView{
		nodes:      make(map[int64]*list.Element),
		cacheQueue: list.New(),
	}

	chainview.setTip(tip)
	return chainview
}

func (c *chainView) tip() *blockNode {
	if c.cacheQueue.Len() == 0 {
		return nil
	}
	elem := c.cacheQueue.Front()
	blocknode := elem.Value.(*blockNode)
	return blocknode
}

func (c *chainView) Tip() *blockNode {
	c.mtx.Lock()
	tip := c.tip()
	c.mtx.Unlock()
	return tip
}

// 插入到list的头，节点超出blockNodeCacheLimit从back开始删除
func (c *chainView) setTip(node *blockNode) {
	//需要检查node节点的父节点是当前的tip节点
	if c.cacheQueue.Len() != 0 {

		if node.parent != c.tip() {
			chainlog.Error("setTip err", "node.height", node.height, "node.hash", common.ToHex(node.hash))
			chainlog.Error("setTip err", "c.tip().height", c.tip().height, "c.tip().hash", common.ToHex(c.tip().hash))
			return
		}
	}
	// Create entry in cache and append to cacheQueue.
	elem := c.cacheQueue.PushFront(node)
	c.nodes[node.height] = elem

	// Maybe expire an item.
	if int64(c.cacheQueue.Len()) > blockNodeCacheLimit {
		height := c.cacheQueue.Remove(c.cacheQueue.Back()).(*blockNode).height
		delete(c.nodes, height)
	}
	chainlog.Debug("setTip", "node.height", node.height, "node.hash", common.ToHex(node.hash))
}

func (c *chainView) SetTip(node *blockNode) {
	c.mtx.Lock()
	c.setTip(node)
	c.mtx.Unlock()
}

// 删除tip节点，主要是节点回退时使用
func (c *chainView) delTip(node *blockNode) {
	if c.tip() != node {
		chainlog.Error("delTip err", "node.height", node.height, "node.hash", node.hash)
		chainlog.Error("delTip err", "tip.height", c.tip().height, "tip.hash", c.tip().hash)
	}

	elem, ok := c.nodes[node.height]
	if ok {
		delheight := c.cacheQueue.Remove(elem).(*blockNode).height
		if delheight != node.height {
			chainlog.Error("delTip height err ", "height", node.height, "delheight", delheight)
		}
		delete(c.nodes, delheight)
	}
}

func (c *chainView) DelTip(node *blockNode) {
	c.mtx.Lock()
	c.delTip(node)
	c.mtx.Unlock()
}

// 返回 chain view tip 的height
func (c *chainView) height() int64 {
	node := c.tip()
	if node != nil {
		return node.height
	}
	return -1
}

func (c *chainView) Height() int64 {
	c.mtx.Lock()
	height := c.height()
	c.mtx.Unlock()
	return height
}

//获取指定height的node
func (c *chainView) nodeByHeight(height int64) *blockNode {
	if height < 0 || height > c.height() {
		return nil
	}

	elem, ok := c.nodes[height]
	if ok {
		return elem.Value.(*blockNode)
	}

	return nil
}

func (c *chainView) NodeByHeight(height int64) *blockNode {
	c.mtx.Lock()
	node := c.nodeByHeight(height)
	c.mtx.Unlock()
	return node
}

// This function is safe for concurrent access.
func (c *chainView) Equals(other *chainView) bool {
	c.mtx.Lock()
	other.mtx.Lock()
	equals := len(c.nodes) == len(other.nodes) && c.tip() == other.tip()
	other.mtx.Unlock()
	c.mtx.Unlock()
	return equals
}

// contains returns whether or not the chain view contains the passed block node.
func (c *chainView) contains(node *blockNode) bool {
	return c.nodeByHeight(node.height) == node
}

func (c *chainView) Contains(node *blockNode) bool {
	c.mtx.Lock()
	contains := c.contains(node)
	c.mtx.Unlock()
	return contains
}

func (c *chainView) next(node *blockNode) *blockNode {
	if node == nil || !c.contains(node) {
		return nil
	}

	return c.nodeByHeight(node.height + 1)
}
func (c *chainView) Next(node *blockNode) *blockNode {
	c.mtx.Lock()
	next := c.next(node)
	c.mtx.Unlock()
	return next
}

// findFork returns the final common block between the provided node and the
// the chain view.  It will return nil if there is no common block.  This only
// differs from the exported version in that it is up to the caller to ensure
// the lock is held.

func (c *chainView) findFork(node *blockNode) *blockNode {

	if node == nil {
		return nil
	}

	chainHeight := c.height()
	if node.height > chainHeight {
		node = node.Ancestor(chainHeight)
	}

	for node != nil && !c.contains(node) {
		node = node.parent
	}

	return node
}

func (c *chainView) FindFork(node *blockNode) *blockNode {
	c.mtx.Lock()
	fork := c.findFork(node)
	c.mtx.Unlock()
	return fork
}

func (c *chainView) HaveBlock(hash []byte, height int64) bool {
	c.mtx.Lock()
	defer c.mtx.Unlock()
	node := c.nodeByHeight(height)
	if node != nil {
		if bytes.Equal(hash, node.hash) {
			return true
		}
	}
	return false
}

func (c *chainView) PrintTip() {
	chainlog.Error("PrintTip info:")
	var next *list.Element
	for e := c.cacheQueue.Front(); e != nil; e = next {
		next = e.Next()
		blocknode := e.Value.(*blockNode)
		chainlog.Error("PrintTip node:", "blocknode.height", blocknode.height, "blocknode.hash", common.ToHex(blocknode.hash))
	}
}
