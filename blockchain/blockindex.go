package blockchain

import (
	"container/list"
	"math/big"
	"sync"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/difficulty"
	"gitlab.33.cn/chain33/chain33/types"
)

type blockNode struct {
	parent     *blockNode
	hash       []byte
	Difficulty *big.Int
	height     int64
	statehash  []byte
	broadcast  bool
}

type blockIndex struct {
	sync.RWMutex
	index      map[string]*list.Element
	cacheQueue *list.List
}

const (
	indexCacheLimit = 500
)

func initBlockNode(node *blockNode, block *types.Block, broadcast bool) {
	*node = blockNode{
		hash:       block.Hash(),
		Difficulty: difficulty.CalcWork(block.Difficulty),
		height:     block.Height,
		statehash:  block.GetStateHash(),
		broadcast:  broadcast,
	}
}

func newBlockNode(broadcast bool, block *types.Block) *blockNode {
	var node blockNode
	initBlockNode(&node, block, broadcast)
	return &node
}
func newPreGenBlockNode() *blockNode {
	node := &blockNode{
		hash:       common.Hash{}.Bytes(),
		Difficulty: big.NewInt(-1),
		height:     -1,
		statehash:  common.Hash{}.Bytes(),
		broadcast:  false,
	}
	return node
}

func (node *blockNode) Ancestor(height int64) *blockNode {
	if height < 0 || height > node.height {
		return nil
	}

	n := node
	for ; n != nil && n.height != height; n = n.parent {
		// Intentionally left blank
	}

	return n
}
func (node *blockNode) RelativeAncestor(distance int64) *blockNode {
	return node.Ancestor(node.height - distance)
}

func newBlockIndex() *blockIndex {
	return &blockIndex{
		index:      make(map[string]*list.Element),
		cacheQueue: list.New(),
	}
}

func (bi *blockIndex) HaveBlock(hash []byte) bool {
	bi.Lock()
	defer bi.Unlock()
	_, hasBlock := bi.index[string(hash)]

	return hasBlock
}

func (bi *blockIndex) LookupNode(hash []byte) *blockNode {
	bi.Lock()
	defer bi.Unlock()

	elem, ok := bi.index[string(hash)]
	if ok {
		return elem.Value.(*blockNode)
	}

	return nil
}

func (bi *blockIndex) AddNode(node *blockNode) {
	bi.Lock()
	defer bi.Unlock()

	// Create entry in cache and append to cacheQueue.
	elem := bi.cacheQueue.PushBack(node)
	bi.index[string(node.hash)] = elem

	// Maybe expire an item.
	if int64(bi.cacheQueue.Len()) > indexCacheLimit {
		hash := bi.cacheQueue.Remove(bi.cacheQueue.Front()).(*blockNode).hash
		delete(bi.index, string(hash))
	}
}
