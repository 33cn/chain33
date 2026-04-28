// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/assert"
)

func TestNewBlockNodeByHeader(t *testing.T) {
	header := &types.Header{
		Hash:       []byte("hash123456789012345678901234567890"),
		Height:     100,
		Difficulty: 5,
		StateHash:  []byte("statehash12345678901234567890123456"),
		BlockTime:  1234567890,
	}
	node := newBlockNodeByHeader(true, header, "peer1", 42)
	assert.Equal(t, "peer1", node.pid)
	assert.Equal(t, int64(42), node.sequence)
	assert.Equal(t, int64(100), node.height)
	assert.Equal(t, int64(1234567890), node.BlockTime)
	assert.True(t, node.broadcast)
}

func TestNewBlockNodeByHeaderNotBroadcast(t *testing.T) {
	header := &types.Header{
		Hash:       []byte("hash223456789012345678901234567890"),
		Height:     50,
		Difficulty: 1,
		StateHash:  []byte("statehash22345678901234567890123456"),
	}
	node := newBlockNodeByHeader(false, header, "peer2", 0)
	assert.False(t, node.broadcast)
	assert.Equal(t, int64(50), node.height)
}

func TestPreGenBlockNode(t *testing.T) {
	node := newPreGenBlockNode()
	assert.Equal(t, int64(-1), node.height)
	assert.Equal(t, "self", node.pid)
	assert.False(t, node.broadcast)
	assert.NotNil(t, node.hash)
	assert.NotNil(t, node.Difficulty)
	assert.Equal(t, int64(-1), node.Difficulty.Int64())
}

func TestBlockNodeAncestor(t *testing.T) {
	// Build a chain: node(10) -> node(9) -> ... -> node(0)
	var nodes []*blockNode
	for i := int64(0); i <= 10; i++ {
		node := &blockNode{height: i}
		nodes = append(nodes, node)
	}
	for i := int64(10); i > 0; i-- {
		nodes[i].parent = nodes[i-1]
	}

	// Test ancestor at same height returns self
	assert.Equal(t, nodes[10], nodes[10].Ancestor(10))

	// Test ancestor at different height
	assert.Equal(t, nodes[5], nodes[10].Ancestor(5))
	assert.Equal(t, nodes[0], nodes[10].Ancestor(0))

	// Test invalid height
	assert.Nil(t, nodes[10].Ancestor(-1))
	assert.Nil(t, nodes[10].Ancestor(11))
	assert.Nil(t, nodes[5].Ancestor(10))
}

func TestBlockNodeRelativeAncestor(t *testing.T) {
	// Build a chain: node(10) -> node(9) -> ... -> node(0)
	var nodes []*blockNode
	for i := int64(0); i <= 10; i++ {
		node := &blockNode{height: i}
		nodes = append(nodes, node)
	}
	for i := int64(10); i > 0; i-- {
		nodes[i].parent = nodes[i-1]
	}

	assert.Equal(t, nodes[8], nodes[10].RelativeAncestor(2))
	assert.Equal(t, nodes[0], nodes[10].RelativeAncestor(10))
}

func TestNewBlockIndex(t *testing.T) {
	bi := newBlockIndex()
	assert.NotNil(t, bi)
	assert.NotNil(t, bi.index)
	assert.NotNil(t, bi.cacheQueue)
}

func TestBlockIndexHaveBlock(t *testing.T) {
	bi := newBlockIndex()
	hash := []byte("test-hash")
	assert.False(t, bi.HaveBlock(hash))

	node := &blockNode{hash: hash, height: 1}
	bi.AddNode(node)
	assert.True(t, bi.HaveBlock(hash))
}

func TestBlockIndexLookupNode(t *testing.T) {
	bi := newBlockIndex()
	hash := []byte("lookup-hash")
	assert.Nil(t, bi.LookupNode(hash))

	node := &blockNode{hash: hash, height: 5}
	bi.AddNode(node)
	found := bi.LookupNode(hash)
	assert.NotNil(t, found)
	assert.Equal(t, int64(5), found.height)
}

func TestBlockIndexAddNode(t *testing.T) {
	bi := newBlockIndex()
	node := &blockNode{hash: []byte("add-hash"), height: 10}
	bi.AddNode(node)
	assert.True(t, bi.HaveBlock([]byte("add-hash")))
}

func TestBlockIndexUpdateNode(t *testing.T) {
	bi := newBlockIndex()
	oldHash := []byte("old-hash")
	node := &blockNode{hash: oldHash, height: 10}
	bi.AddNode(node)

	newHash := []byte("new-hash")
	newNode := &blockNode{hash: newHash, height: 10}
	assert.True(t, bi.UpdateNode(oldHash, newNode))
	assert.False(t, bi.HaveBlock(oldHash))
	assert.True(t, bi.HaveBlock(newHash))

	// Update non-existent node
	assert.False(t, bi.UpdateNode([]byte("no-such"), newNode))
}

func TestBlockIndexDelNode(t *testing.T) {
	bi := newBlockIndex()
	hash := []byte("del-hash")
	node := &blockNode{hash: hash, height: 10}
	bi.AddNode(node)
	assert.True(t, bi.HaveBlock(hash))

	bi.DelNode(hash)
	assert.False(t, bi.HaveBlock(hash))

	// Delete non-existent should not panic
	bi.DelNode([]byte("no-such"))
}

func TestBlockIndexCacheLimit(t *testing.T) {
	bi := newBlockIndex()
	// Add many nodes to trigger cache expiration
	for i := 0; i < 10; i++ {
		hash := []byte{byte(i)}
		node := &blockNode{hash: hash, height: int64(i)}
		bi.AddNode(node)
	}
	// All recently added should be present
	for i := 0; i < 10; i++ {
		hash := []byte{byte(i)}
		assert.True(t, bi.HaveBlock(hash), "hash %d should exist", i)
	}
}
