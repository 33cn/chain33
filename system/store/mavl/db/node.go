// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"bytes"

	"fmt"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/types"
	farm "github.com/dgryski/go-farm"
	"github.com/golang/protobuf/proto"
)

// Node merkle avl Node
type Node struct {
	key        []byte
	value      []byte
	height     int32
	size       int32
	hash       []byte
	leftHash   []byte
	leftNode   *Node
	rightHash  []byte
	rightNode  *Node
	parentNode *Node
	persisted  bool
}

// NewNode 创建节点；保存数据的是叶子节点
func NewNode(key []byte, value []byte) *Node {
	return &Node{
		key:    key,
		value:  value,
		height: 0,
		size:   1,
	}
}

// MakeNode 从数据库中读取数据，创建Node
// NOTE: The hash is not saved or set.  The caller should set the hash afterwards.
// (Presumably the caller already has the hash)
func MakeNode(buf []byte, t *Tree) (node *Node, err error) {
	node = &Node{}

	var storeNode types.StoreNode

	err = proto.Unmarshal(buf, &storeNode)
	if err != nil {
		return nil, err
	}

	// node header
	node.height = storeNode.Height
	node.size = storeNode.Size
	node.key = storeNode.Key

	//leaf(叶子节点保存数据)
	if node.height == 0 {
		node.value = storeNode.Value
	} else {
		node.leftHash = storeNode.LeftHash
		node.rightHash = storeNode.RightHash
	}
	return node, nil
}

func (node *Node) _copy() *Node {
	if node.height == 0 {
		panic("Why are you copying a value node?")
	}
	return &Node{
		key:        node.key,
		height:     node.height,
		size:       node.size,
		hash:       nil, // Going to be mutated anyways.
		leftHash:   node.leftHash,
		leftNode:   node.leftNode,
		rightHash:  node.rightHash,
		rightNode:  node.rightNode,
		parentNode: node.parentNode,
		persisted:  false, // Going to be mutated, so it can't already be persisted.
	}
}

func (node *Node) has(t *Tree, key []byte) (has bool) {
	if bytes.Equal(node.key, key) {
		return true
	}
	if node.height == 0 {
		return false
	}
	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(t).has(t, key)
	}
	return node.getRightNode(t).has(t, key)
}

func (node *Node) get(t *Tree, key []byte) (index int32, value []byte, exists bool) {
	if node.height == 0 {
		cmp := bytes.Compare(node.key, key)
		if cmp == 0 {
			return 0, node.value, true
		} else if cmp == -1 {
			return 1, nil, false
		} else {
			return 0, nil, false
		}
	}
	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(t).get(t, key)
	}
	rightNode := node.getRightNode(t)
	index, value, exists = rightNode.get(t, key)
	index += node.size - rightNode.size
	return index, value, exists
}

func (node *Node) getHash(t *Tree, key []byte) (index int32, hash []byte, exists bool) {
	if node.height == 0 {
		cmp := bytes.Compare(node.key, key)
		if cmp == 0 {
			return 0, node.hash, true
		} else if cmp == -1 {
			return 1, nil, false
		} else {
			return 0, nil, false
		}
	}
	if bytes.Compare(key, node.key) < 0 {
		return node.getLeftNode(t).getHash(t, key)
	}
	rightNode := node.getRightNode(t)
	index, hash, exists = rightNode.getHash(t, key)
	index += node.size - rightNode.size
	return index, hash, exists
}

//通过index获取leaf节点信息
func (node *Node) getByIndex(t *Tree, index int32) (key []byte, value []byte) {
	if node.height == 0 {
		if index == 0 {
			return node.key, node.value
		}
		panic("getByIndex asked for invalid index")
	} else {
		// TODO: could improve this by storing the sizes as well as left/right hash.
		leftNode := node.getLeftNode(t)
		if index < leftNode.size {
			return leftNode.getByIndex(t, index)
		}
		return node.getRightNode(t).getByIndex(t, index-leftNode.size)
	}
}

// Hash 计算节点的hash
func (node *Node) Hash(t *Tree) []byte {
	if node.hash != nil {
		return node.hash
	}

	//leafnode
	if node.height == 0 {
		var leafnode types.LeafNode
		leafnode.Height = node.height
		leafnode.Key = node.key
		leafnode.Size = node.size
		leafnode.Value = node.value
		node.hash = leafnode.Hash()

		if enableMavlPrefix && node.height != t.root.height {
			hashKey := genPrefixHashKey(node, t.blockHeight)
			hashKey = append(hashKey, node.hash...)
			node.hash = hashKey
		}
	} else {
		var innernode types.InnerNode
		innernode.Height = node.height
		innernode.Size = node.size

		// left
		if node.leftNode != nil {
			leftHash := node.leftNode.Hash(t)
			node.leftHash = leftHash
		}
		if node.leftHash == nil {
			panic("node.leftHash was nil in writeHashBytes")
		}
		innernode.LeftHash = node.leftHash

		// right
		if node.rightNode != nil {
			rightHash := node.rightNode.Hash(t)
			node.rightHash = rightHash
		}
		if node.rightHash == nil {
			panic("node.rightHash was nil in writeHashBytes")
		}
		innernode.RightHash = node.rightHash
		node.hash = innernode.Hash()
		if enableMavlPrefix && node.height != t.root.height {
			hashKey := genPrefixHashKey(node, t.blockHeight)
			hashKey = append(hashKey, node.hash...)
			node.hash = hashKey
		}

		if enablePrune {
			//加入parentNode
			if node.leftNode != nil && node.leftNode.height != t.root.height {
				node.leftNode.parentNode = node
			}
			if node.rightNode != nil && node.rightNode.height != t.root.height {
				node.rightNode.parentNode = node
			}
		}
	}

	if enableMemTree {
		updateLocalMemTree(t, node)
	}
	return node.hash
}

// NOTE: clears leftNode/rigthNode recursively sets hashes recursively
func (node *Node) save(t *Tree) int64 {
	if node.hash == nil {
		node.hash = node.Hash(t)
	}
	if node.persisted {
		return 0
	}
	var leftsaveNodeNo int64
	var rightsaveNodeNo int64

	// save children
	if node.leftNode != nil {
		leftsaveNodeNo = node.leftNode.save(t)
		node.leftNode = nil
	}
	if node.rightNode != nil {
		rightsaveNodeNo = node.rightNode.save(t)
		node.rightNode = nil
	}

	// save node
	t.ndb.SaveNode(t, node)
	return leftsaveNodeNo + rightsaveNodeNo + 1
}

// 保存root节点hash以及区块高度
func (node *Node) saveRootHash(t *Tree) (err error) {
	if node.hash == nil || t.ndb == nil || t.ndb.db == nil {
		return
	}
	h := &types.Int64{}
	h.Data = t.blockHeight
	value, err := proto.Marshal(h)
	if err != nil {
		return err
	}
	t.ndb.batch.Set(genRootHashHeight(t.blockHeight, node.hash), value)
	return nil
}

//将内存中的node转换成存储到db中的格式
func (node *Node) storeNode(t *Tree) []byte {
	var storeNode types.StoreNode

	// node header
	storeNode.Height = node.height
	storeNode.Size = node.size
	storeNode.Key = node.key
	storeNode.Value = nil
	storeNode.LeftHash = nil
	storeNode.RightHash = nil

	//leafnode
	if node.height == 0 {
		if !enableMvcc {
			storeNode.Value = node.value
		}
	} else {
		// left
		if node.leftHash == nil {
			panic("node.leftHash was nil in writePersistBytes")
		}
		storeNode.LeftHash = node.leftHash

		// right
		if node.rightHash == nil {
			panic("node.rightHash was nil in writePersistBytes")
		}
		storeNode.RightHash = node.rightHash
	}
	storeNodebytes, err := proto.Marshal(&storeNode)
	if err != nil {
		panic(err)
	}
	return storeNodebytes
}

//从指定node开始插入一个新的node，updated表示是否有叶子结点的value更新
func (node *Node) set(t *Tree, key []byte, value []byte) (newSelf *Node, updated bool) {
	if node.height == 0 {
		cmp := bytes.Compare(key, node.key)
		if cmp < 0 {
			return &Node{
				key:       node.key,
				height:    1,
				size:      2,
				leftNode:  NewNode(key, value),
				rightNode: node,
			}, false
		} else if cmp == 0 {
			removeOrphan(t, node)
			return NewNode(key, value), true
		} else {
			return &Node{
				key:       key,
				height:    1,
				size:      2,
				leftNode:  node,
				rightNode: NewNode(key, value),
			}, false
		}
	} else {
		removeOrphan(t, node)
		node = node._copy()
		if bytes.Compare(key, node.key) < 0 {
			node.leftNode, updated = node.getLeftNode(t).set(t, key, value)
			node.leftHash = nil // leftHash is yet unknown
		} else {
			node.rightNode, updated = node.getRightNode(t).set(t, key, value)
			node.rightHash = nil // rightHash is yet unknown
		}
		if updated {
			return node, updated
		}
		//有节点插入，需要重新计算height和size以及tree的平衡
		node.calcHeightAndSize(t)
		return node.balance(t), updated
	}
}

func (node *Node) getLeftNode(t *Tree) *Node {
	if node.leftNode != nil {
		return node.leftNode
	}
	leftNode, err := t.ndb.GetNode(t, node.leftHash)
	if err != nil {
		panic(fmt.Sprintln("left hash", common.ToHex(node.leftHash), err)) //数据库已经损坏
	}
	return leftNode
}

func (node *Node) getRightNode(t *Tree) *Node {
	if node.rightNode != nil {
		return node.rightNode
	}
	rightNode, err := t.ndb.GetNode(t, node.rightHash)
	if err != nil {
		panic(fmt.Sprintln("right hash", common.ToHex(node.rightHash), err))
	}
	return rightNode
}

// NOTE: overwrites node TODO: optimize balance & rotate
func (node *Node) rotateRight(t *Tree) *Node {
	node = node._copy()
	l := node.getLeftNode(t)
	removeOrphan(t, l)
	_l := l._copy()

	_lrHash, _lrCached := _l.rightHash, _l.rightNode
	_l.rightHash, _l.rightNode = node.hash, node
	node.leftHash, node.leftNode = _lrHash, _lrCached

	node.calcHeightAndSize(t)
	_l.calcHeightAndSize(t)

	return _l
}

// NOTE: overwrites node TODO: optimize balance & rotate
func (node *Node) rotateLeft(t *Tree) *Node {
	node = node._copy()
	r := node.getRightNode(t)
	removeOrphan(t, r)
	_r := r._copy()

	_rlHash, _rlCached := _r.leftHash, _r.leftNode
	_r.leftHash, _r.leftNode = node.hash, node
	node.rightHash, node.rightNode = _rlHash, _rlCached

	node.calcHeightAndSize(t)
	_r.calcHeightAndSize(t)

	return _r
}

// NOTE: mutates height and size
func (node *Node) calcHeightAndSize(t *Tree) {
	leftN := node.getLeftNode(t)
	rightN := node.getRightNode(t)
	node.height = maxInt32(leftN.height, rightN.height) + 1
	node.size = leftN.size + rightN.size
}

func (node *Node) calcBalance(t *Tree) int {
	return int(node.getLeftNode(t).height) - int(node.getRightNode(t).height)
}

// NOTE: assumes that node can be modified TODO: optimize balance & rotate
func (node *Node) balance(t *Tree) (newSelf *Node) {
	if node.persisted {
		panic("Unexpected balance() call on persisted node")
	}
	balance := node.calcBalance(t)
	if balance > 1 {
		if node.getLeftNode(t).calcBalance(t) >= 0 {
			// Left Left Case
			return node.rotateRight(t)
		}
		// Left Right Case
		// node = node._copy()
		left := node.getLeftNode(t)
		removeOrphan(t, left)
		node.leftHash, node.leftNode = nil, left.rotateLeft(t)
		//node.calcHeightAndSize()
		return node.rotateRight(t)
	}
	if balance < -1 {
		if node.getRightNode(t).calcBalance(t) <= 0 {
			// Right Right Case
			return node.rotateLeft(t)
		}
		// Right Left Case
		// node = node._copy()
		right := node.getRightNode(t)
		removeOrphan(t, right)
		node.rightHash, node.rightNode = nil, right.rotateRight(t)
		//node.calcHeightAndSize()
		return node.rotateLeft(t)
	}
	// Nothing changed
	return node
}

// newHash/newNode: The new hash or node to replace node after remove.
// newKey: new leftmost leaf key for tree after successfully removing 'key' if changed.
// value: removed value.
func (node *Node) remove(t *Tree, key []byte) (
	newHash []byte, newNode *Node, newKey []byte, value []byte, removed bool) {
	if node.height == 0 {
		if bytes.Equal(key, node.key) {
			removeOrphan(t, node)
			return nil, nil, nil, node.value, true
		}
		return node.hash, node, nil, nil, false
	}
	if bytes.Compare(key, node.key) < 0 {
		var newLeftHash []byte
		var newLeftNode *Node
		newLeftHash, newLeftNode, newKey, value, removed = node.getLeftNode(t).remove(t, key)
		if !removed {
			return node.hash, node, nil, value, false
		} else if newLeftHash == nil && newLeftNode == nil { // left node held value, was removed
			return node.rightHash, node.rightNode, node.key, value, true
		}
		removeOrphan(t, node)
		node = node._copy()
		node.leftHash, node.leftNode = newLeftHash, newLeftNode
		node.calcHeightAndSize(t)
		node = node.balance(t)
		return node.hash, node, newKey, value, true
	}
	var newRightHash []byte
	var newRightNode *Node
	newRightHash, newRightNode, newKey, value, removed = node.getRightNode(t).remove(t, key)
	if !removed {
		return node.hash, node, nil, value, false
	} else if newRightHash == nil && newRightNode == nil { // right node held value, was removed
		return node.leftHash, node.leftNode, nil, value, true
	}
	removeOrphan(t, node)
	node = node._copy()
	node.rightHash, node.rightNode = newRightHash, newRightNode
	if newKey != nil {
		node.key = newKey
	}
	node.calcHeightAndSize(t)
	node = node.balance(t)
	return node.hash, node, nil, value, true
}

func removeOrphan(t *Tree, node *Node) {
	if !node.persisted {
		return
	}
	if t.ndb == nil {
		return
	}
	if enableMemTree && t != nil {
		t.obsoleteNode[uintkey(farm.Hash64(node.hash))] = struct{}{}
	}
	t.ndb.RemoveNode(t, node)
}

// 迭代整个树
func (node *Node) traverse(t *Tree, ascending bool, cb func(*Node) bool) bool {
	return node.traverseInRange(t, nil, nil, ascending, false, 0, func(node *Node, depth uint8) bool {
		return cb(node)
	})
}

func (node *Node) traverseWithDepth(t *Tree, ascending bool, cb func(*Node, uint8) bool) bool {
	return node.traverseInRange(t, nil, nil, ascending, false, 0, cb)
}

func (node *Node) traverseInRange(t *Tree, start, end []byte, ascending bool, inclusive bool, depth uint8, cb func(*Node, uint8) bool) bool {
	afterStart := start == nil || bytes.Compare(start, node.key) <= 0
	beforeEnd := end == nil || bytes.Compare(node.key, end) < 0
	if inclusive {
		beforeEnd = end == nil || bytes.Compare(node.key, end) <= 0
	}

	stop := false
	if afterStart && beforeEnd {
		// IterateRange ignores this if not leaf
		stop = cb(node, depth)
	}
	if stop {
		return stop
	}
	if node.height == 0 {
		return stop
	}

	if ascending {
		// check lower nodes, then higher
		if afterStart {
			stop = node.getLeftNode(t).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if beforeEnd {
			stop = node.getRightNode(t).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
	} else {
		// check the higher nodes first
		if beforeEnd {
			stop = node.getRightNode(t).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
		if stop {
			return stop
		}
		if afterStart {
			stop = node.getLeftNode(t).traverseInRange(t, start, end, ascending, inclusive, depth+1, cb)
		}
	}

	return stop
}

// Only used in testing...
func (node *Node) lmd(t *Tree) *Node {
	if node.height == 0 {
		return node
	}
	return node.getLeftNode(t).lmd(t)
}

// Only used in testing...
func (node *Node) rmd(t *Tree) *Node {
	if node.height == 0 {
		return node
	}
	return node.getRightNode(t).rmd(t)
}
