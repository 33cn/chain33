package mavl

import (
	"bytes"

	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var nodelog = log.New("module", "mptnode")

// merkle avl Node
type MAVLNode struct {
	key       []byte
	value     []byte
	height    int32
	size      int32
	hash      []byte
	leftHash  []byte
	leftNode  *MAVLNode
	rightHash []byte
	rightNode *MAVLNode
	persisted bool
}

func NewMAVLNode(key []byte, value []byte) *MAVLNode {
	return &MAVLNode{
		key:    key,
		value:  value,
		height: 0,
		size:   1,
	}
}

// NOTE: The hash is not saved or set.  The caller should set the hash afterwards.
// (Presumably the caller already has the hash)
func MakeMAVLNode(buf []byte, t *MAVLTree) (node *MAVLNode, err error) {
	node = &MAVLNode{}

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

func (node *MAVLNode) _copy() *MAVLNode {
	if node.height == 0 {
		panic("Why are you copying a value node?")
	}
	return &MAVLNode{
		key:       node.key,
		height:    node.height,
		size:      node.size,
		hash:      nil, // Going to be mutated anyways.
		leftHash:  node.leftHash,
		leftNode:  node.leftNode,
		rightHash: node.rightHash,
		rightNode: node.rightNode,
		persisted: false, // Going to be mutated, so it can't already be persisted.
	}
}

func (node *MAVLNode) has(t *MAVLTree, key []byte) (has bool) {
	if bytes.Compare(node.key, key) == 0 {
		return true
	}
	if node.height == 0 {
		return false
	} else {
		if bytes.Compare(key, node.key) < 0 {
			return node.getLeftNode(t).has(t, key)
		} else {
			return node.getRightNode(t).has(t, key)
		}
	}
}

func (node *MAVLNode) loadCache(t *MAVLTree, stopHeight int) {
	if node.height == 0 || stopHeight <= 0 {
		return
	} else {
		node.getLeftNode(t).loadCache(t, stopHeight-1)
		node.getRightNode(t).loadCache(t, stopHeight-1)
	}
}

func (node *MAVLNode) get(t *MAVLTree, key []byte) (index int32, value []byte, exists bool) {
	if node.height == 0 {
		cmp := bytes.Compare(node.key, key)
		if cmp == 0 {
			return 0, node.value, true
		} else if cmp == -1 {
			return 1, nil, false
		} else {
			return 0, nil, false
		}
	} else {
		if bytes.Compare(key, node.key) < 0 {
			return node.getLeftNode(t).get(t, key)
		} else {
			rightNode := node.getRightNode(t)
			index, value, exists = rightNode.get(t, key)
			index += node.size - rightNode.size
			return index, value, exists
		}
	}
}

//通过index获取leaf节点信息
func (node *MAVLNode) getByIndex(t *MAVLTree, index int32) (key []byte, value []byte) {
	if node.height == 0 {
		if index == 0 {
			return node.key, node.value
		} else {
			panic("getByIndex asked for invalid index")
		}
	} else {
		// TODO: could improve this by storing the sizes as well as left/right hash.
		leftNode := node.getLeftNode(t)
		if index < leftNode.size {
			return leftNode.getByIndex(t, index)
		} else {
			return node.getRightNode(t).getByIndex(t, index-leftNode.size)
		}
	}
}

// 计算节点的hash
func (node *MAVLNode) Hash(t *MAVLTree) []byte {
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
	}

	return node.hash
}

// NOTE: clears leftNode/rigthNode recursively sets hashes recursively
func (node *MAVLNode) save(t *MAVLTree) {
	if node.hash == nil {
		node.hash = node.Hash(t)
	}
	if node.persisted {
		return
	}

	// save children
	if node.leftNode != nil {
		node.leftNode.save(t)
		node.leftNode = nil
	}
	if node.rightNode != nil {
		node.rightNode.save(t)
		node.rightNode = nil
	}

	// save node
	t.batch.SaveNode(t, node)
	return
}

//将内存中的node转换成存储到db中的格式
func (node *MAVLNode) storeNode(t *MAVLTree) []byte {
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
		storeNode.Value = node.value
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
func (node *MAVLNode) set(t *MAVLTree, key []byte, value []byte) (newSelf *MAVLNode, updated bool) {
	if node.height == 0 {
		cmp := bytes.Compare(key, node.key)
		if cmp < 0 {
			return &MAVLNode{
				key:       node.key,
				height:    1,
				size:      2,
				leftNode:  NewMAVLNode(key, value),
				rightNode: node,
			}, false
		} else if cmp == 0 {
			return NewMAVLNode(key, value), true
		} else {
			return &MAVLNode{
				key:       key,
				height:    1,
				size:      2,
				leftNode:  node,
				rightNode: NewMAVLNode(key, value),
			}, false
		}
	} else {
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
		} else { //有节点插入，需要重新计算height和size以及tree的平衡
			node.calcHeightAndSize(t)
			return node.balance(t), updated
		}
	}
}

func (node *MAVLNode) getLeftNode(t *MAVLTree) *MAVLNode {
	if node.leftNode != nil {
		return node.leftNode
	} else {
		leftNode, err := t.ndb.GetNode(t, node.leftHash)
		if err != nil {
			panic(err) //数据库已经损坏
		}
		return leftNode
	}
}

func (node *MAVLNode) getRightNode(t *MAVLTree) *MAVLNode {
	if node.rightNode != nil {
		return node.rightNode
	} else {
		rightNode, err := t.ndb.GetNode(t, node.rightHash)
		if err != nil {
			panic(err)
		}
		return rightNode
	}
}

// NOTE: overwrites node TODO: optimize balance & rotate
func (node *MAVLNode) rotateRight(t *MAVLTree) *MAVLNode {
	node = node._copy()
	l := node.getLeftNode(t)
	_l := l._copy()

	_lrHash, _lrCached := _l.rightHash, _l.rightNode
	_l.rightHash, _l.rightNode = node.hash, node
	node.leftHash, node.leftNode = _lrHash, _lrCached

	node.calcHeightAndSize(t)
	_l.calcHeightAndSize(t)

	return _l
}

// NOTE: overwrites node TODO: optimize balance & rotate
func (node *MAVLNode) rotateLeft(t *MAVLTree) *MAVLNode {
	node = node._copy()
	r := node.getRightNode(t)
	_r := r._copy()

	_rlHash, _rlCached := _r.leftHash, _r.leftNode
	_r.leftHash, _r.leftNode = node.hash, node
	node.rightHash, node.rightNode = _rlHash, _rlCached

	node.calcHeightAndSize(t)
	_r.calcHeightAndSize(t)

	return _r
}

// NOTE: mutates height and size
func (node *MAVLNode) calcHeightAndSize(t *MAVLTree) {
	node.height = maxInt32(node.getLeftNode(t).height, node.getRightNode(t).height) + 1
	node.size = node.getLeftNode(t).size + node.getRightNode(t).size
}

func (node *MAVLNode) calcBalance(t *MAVLTree) int {
	return int(node.getLeftNode(t).height) - int(node.getRightNode(t).height)
}

// NOTE: assumes that node can be modified TODO: optimize balance & rotate
func (node *MAVLNode) balance(t *MAVLTree) (newSelf *MAVLNode) {
	if node.persisted {
		panic("Unexpected balance() call on persisted node")
	}
	balance := node.calcBalance(t)
	if balance > 1 {
		if node.getLeftNode(t).calcBalance(t) >= 0 {
			// Left Left Case
			return node.rotateRight(t)
		} else {
			// Left Right Case
			// node = node._copy()
			left := node.getLeftNode(t)
			node.leftHash, node.leftNode = nil, left.rotateLeft(t)
			//node.calcHeightAndSize()
			return node.rotateRight(t)
		}
	}
	if balance < -1 {
		if node.getRightNode(t).calcBalance(t) <= 0 {
			// Right Right Case
			return node.rotateLeft(t)
		} else {
			// Right Left Case
			// node = node._copy()
			right := node.getRightNode(t)
			node.rightHash, node.rightNode = nil, right.rotateRight(t)
			//node.calcHeightAndSize()
			return node.rotateLeft(t)
		}
	}
	// Nothing changed
	return node
}

// newHash/newNode: The new hash or node to replace node after remove.
// newKey: new leftmost leaf key for tree after successfully removing 'key' if changed.
// value: removed value.
func (node *MAVLNode) remove(t *MAVLTree, key []byte) (
	newHash []byte, newNode *MAVLNode, newKey []byte, value []byte, removed bool) {
	if node.height == 0 {
		if bytes.Compare(key, node.key) == 0 {
			removeOrphan(t, node)
			return nil, nil, nil, node.value, true
		} else {
			return node.hash, node, nil, nil, false
		}
	} else {
		if bytes.Compare(key, node.key) < 0 {
			var newLeftHash []byte
			var newLeftNode *MAVLNode
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
		} else {
			var newRightHash []byte
			var newRightNode *MAVLNode
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
	}
}

func removeOrphan(t *MAVLTree, node *MAVLNode) {
	if !node.persisted {
		return
	}
	if t.ndb == nil {
		return
	}
}

// 迭代整个树
func (node *MAVLNode) traverse(t *MAVLTree, ascending bool, cb func(*MAVLNode) bool) bool {
	return node.traverseInRange(t, nil, nil, ascending, false, 0, func(node *MAVLNode, depth uint8) bool {
		return cb(node)
	})
}

func (node *MAVLNode) traverseWithDepth(t *MAVLTree, ascending bool, cb func(*MAVLNode, uint8) bool) bool {
	return node.traverseInRange(t, nil, nil, ascending, false, 0, cb)
}

func (node *MAVLNode) traverseInRange(t *MAVLTree, start, end []byte, ascending bool, inclusive bool, depth uint8, cb func(*MAVLNode, uint8) bool) bool {
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
func (node *MAVLNode) lmd(t *MAVLTree) *MAVLNode {
	if node.height == 0 {
		return node
	}
	return node.getLeftNode(t).lmd(t)
}

// Only used in testing...
func (node *MAVLNode) rmd(t *MAVLTree) *MAVLNode {
	if node.height == 0 {
		return node
	}
	return node.getRightNode(t).rmd(t)
}
