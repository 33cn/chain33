// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mavl 默克尔平衡树算法实现以及裁剪
package mavl

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/golang-lru"
)

const (
	hashNodePrefix = "_mh_"
	leafNodePrefix = "_mb_"
)

var (
	// ErrNodeNotExist node is not exist
	ErrNodeNotExist  = errors.New("ErrNodeNotExist")
	treelog          = log.New("module", "mavl")
	emptyRoot        [32]byte
	enableMavlPrefix bool
	// 是否开启MVCC
	enableMvcc bool
)

// EnableMavlPrefix 使能mavl加前缀
func EnableMavlPrefix(enable bool) {
	enableMavlPrefix = enable
}

// EnableMVCC 使能MVCC
func EnableMVCC(enable bool) {
	enableMvcc = enable
}

// Tree merkle avl tree
type Tree struct {
	root *Node
	ndb  *nodeDB
	//batch *nodeBatch
	blockHeight int64
}

// NewTree 新建一个merkle avl 树
func NewTree(db dbm.DB, sync bool) *Tree {
	if db == nil {
		// In-memory IAVLTree
		return &Tree{}
	}
	// Persistent IAVLTree
	ndb := newNodeDB(db, sync)
	return &Tree{
		ndb: ndb,
	}
}

// Copy copy tree
func (t *Tree) Copy() *Tree {
	if t.root == nil {
		return &Tree{
			root: nil,
			ndb:  t.ndb,
		}
	}
	if t.ndb != nil && !t.root.persisted {
		panic("It is unsafe to Copy() an unpersisted tree.")
	} else if t.ndb == nil && t.root.hash == nil {
		t.root.Hash(t)
	}
	return &Tree{
		root: t.root,
		ndb:  t.ndb,
	}
}

// Size 获取tree的叶子节点数
func (t *Tree) Size() int32 {
	if t.root == nil {
		return 0
	}
	return t.root.size
}

// Height 获取tree的高度
func (t *Tree) Height() int32 {
	if t.root == nil {
		return 0
	}
	return t.root.height
}

// Has 判断key是否存在tree中
func (t *Tree) Has(key []byte) bool {
	if t.root == nil {
		return false
	}
	return t.root.has(t, key)
}

// Set 设置k:v pair到tree中
func (t *Tree) Set(key []byte, value []byte) (updated bool) {
	//treelog.Info("IAVLTree.Set", "key", key, "value",value)
	if t.root == nil {
		t.root = NewNode(key, value)
		return false
	}
	t.root, updated = t.root.set(t, key, value)
	return updated
}

// Hash 计算tree 的roothash
func (t *Tree) Hash() []byte {
	if t.root == nil {
		return nil
	}
	hash := t.root.Hash(t)
	return hash
}

// Save 保存整个tree的节点信息到db中
func (t *Tree) Save() []byte {
	if t.root == nil {
		return nil
	}
	if t.ndb != nil {
		saveNodeNo := t.root.save(t)
		treelog.Debug("Tree.Save", "saveNodeNo", saveNodeNo, "tree height", t.blockHeight)
		err := t.ndb.Commit()
		if err != nil {
			return nil
		}
		// 该线程应只允许一个
		if enablePrune && !isPruning() &&
			t.blockHeight%int64(pruneHeight) == 0 &&
			t.blockHeight/int64(pruneHeight) > 1 {
			go pruningTree(t.ndb.db, t.blockHeight)
		}
	}
	return t.root.hash
}

// Load 从db中加载rootnode
func (t *Tree) Load(hash []byte) (err error) {
	if hash == nil {
		return
	}
	if !bytes.Equal(hash, emptyRoot[:]) {
		t.root, err = t.ndb.GetNode(t, hash)
		return err
	}
	return nil
}

// SetBlockHeight 设置block高度到tree
func (t *Tree) SetBlockHeight(height int64) {
	t.blockHeight = height
}

// Get 通过key获取leaf节点信息
func (t *Tree) Get(key []byte) (index int32, value []byte, exists bool) {
	if t.root == nil {
		return 0, nil, false
	}
	return t.root.get(t, key)
}

// GetByIndex 通过index获取leaf节点信息
func (t *Tree) GetByIndex(index int32) (key []byte, value []byte) {
	if t.root == nil {
		return nil, nil
	}
	return t.root.getByIndex(t, index)
}

// Proof 获取指定k:v pair的proof证明
func (t *Tree) Proof(key []byte) (value []byte, proofBytes []byte, exists bool) {
	value, proof := t.ConstructProof(key)
	if proof == nil {
		return nil, nil, false
	}
	var mavlproof types.MAVLProof
	mavlproof.InnerNodes = proof.InnerNodes
	proofBytes, err := proto.Marshal(&mavlproof)
	if err != nil {
		treelog.Error("Proof proto.Marshal err!", "err", err)
		return nil, nil, false
	}
	return value, proofBytes, true
}

// Remove 删除key对应的节点
func (t *Tree) Remove(key []byte) (value []byte, removed bool) {
	if t.root == nil {
		return nil, false
	}
	newRootHash, newRoot, _, value, removed := t.root.remove(t, key)
	if !removed {
		return nil, false
	}
	if newRoot == nil && newRootHash != nil {
		root, err := t.ndb.GetNode(t, newRootHash)
		if err != nil {
			panic(err) //数据库已经损坏
		}
		t.root = root
	} else {
		t.root = newRoot
	}
	return value, true
}

// Iterate 依次迭代遍历树的所有键
func (t *Tree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverse(t, true, func(node *Node) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		}
		return false
	})
}

// IterateRange 在start和end之间的键进行迭代回调[start, end)
func (t *Tree) IterateRange(start, end []byte, ascending bool, fn func(key []byte, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverseInRange(t, start, end, ascending, false, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		}
		return false
	})
}

// IterateRangeInclusive 在start和end之间的键进行迭代回调[start, end]
func (t *Tree) IterateRangeInclusive(start, end []byte, ascending bool, fn func(key, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverseInRange(t, start, end, ascending, true, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		}
		return false
	})
}

//-----------------------------------------------------------------------------

type nodeDB struct {
	mtx     sync.Mutex
	cache   *lru.ARCCache
	db      dbm.DB
	batch   dbm.Batch
	orphans map[string]struct{}
}

type nodeBatch struct {
	batch dbm.Batch
}

func newNodeDB(db dbm.DB, sync bool) *nodeDB {
	ndb := &nodeDB{
		cache:   db.GetCache(),
		db:      db,
		batch:   db.NewBatch(sync),
		orphans: make(map[string]struct{}),
	}
	return ndb
}

// GetNode 从数据库中查询Node
func (ndb *nodeDB) GetNode(t *Tree, hash []byte) (*Node, error) {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	// Check the cache.

	if ndb.cache != nil {
		elem, ok := ndb.cache.Get(string(hash))
		if ok {
			return elem.(*Node), nil
		}
	}
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
	node.persisted = true
	ndb.cacheNode(node)
	return node, nil
}

// GetBatch get db batch handle
func (ndb *nodeDB) GetBatch(sync bool) *nodeBatch {
	return &nodeBatch{ndb.db.NewBatch(sync)}
}

// SaveNode 保存节点
func (ndb *nodeDB) SaveNode(t *Tree, node *Node) {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	if node.hash == nil {
		panic("Expected to find node.hash, but none found.")
	}
	if node.persisted {
		panic("Shouldn't be calling save on an already persisted node.")
	}
	// Save node bytes to db
	storenode := node.storeNode(t)
	ndb.batch.Set(node.hash, storenode)
	if enablePrune && node.height == 0 {
		//save leafnode key&hash
		k := genLeafCountKey(node.key, node.hash, t.blockHeight)
		data := &types.PruneData{
			Height: t.blockHeight,
			Lenth:  int32(len(node.hash)),
		}
		v, err := proto.Marshal(data)
		if err != nil {
			panic(err)
		}
		ndb.batch.Set(k, v)
	}
	node.persisted = true
	ndb.cacheNode(node)
	delete(ndb.orphans, string(node.hash))
	//treelog.Debug("SaveNode", "hash", node.hash, "height", node.height, "value", node.value)
}

//cache缓存节点
func (ndb *nodeDB) cacheNode(node *Node) {
	//接进叶子节点，不容易命中cache，就不做cache
	if ndb.cache != nil && node.height > 2 {
		ndb.cache.Add(string(node.hash), node)
		if ndb.cache.Len()%10000 == 0 {
			log.Info("store db cache ", "len", ndb.cache.Len())
		}
	}
}

// RemoveNode 删除节点
func (ndb *nodeDB) RemoveNode(t *Tree, node *Node) {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	if node.hash == nil {
		panic("Expected to find node.hash, but none found.")
	}
	if !node.persisted {
		panic("Shouldn't be calling remove on a non-persisted node.")
	}
	if ndb.cache != nil {
		ndb.cache.Remove(string(node.hash))
	}
	ndb.orphans[string(node.hash)] = struct{}{}
}

// Commit 提交状态tree，批量写入db中
func (ndb *nodeDB) Commit() error {

	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	// Write saves
	err := ndb.batch.Write()
	if err != nil {
		treelog.Error("Commit batch.Write err", "err", err)
	}

	ndb.batch = nil
	ndb.orphans = make(map[string]struct{})
	return err
}

// SetKVPair 设置kv对外接口
func SetKVPair(db dbm.DB, storeSet *types.StoreSet, sync bool) ([]byte, error) {
	tree := NewTree(db, sync)
	tree.SetBlockHeight(storeSet.Height)
	err := tree.Load(storeSet.StateHash)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(storeSet.KV); i++ {
		tree.Set(storeSet.KV[i].Key, storeSet.KV[i].Value)
	}
	return tree.Save(), nil
}

// GetKVPair 获取kv对外接口
func GetKVPair(db dbm.DB, storeGet *types.StoreGet) ([][]byte, error) {
	tree := NewTree(db, true)
	err := tree.Load(storeGet.StateHash)
	values := make([][]byte, len(storeGet.Keys))
	if err != nil {
		return values, err
	}
	for i := 0; i < len(storeGet.Keys); i++ {
		_, value, exit := tree.Get(storeGet.Keys[i])
		if exit {
			values[i] = value
		}
	}
	return values, nil
}

// GetKVPairProof 获取指定k:v pair的proof证明
func GetKVPairProof(db dbm.DB, roothash []byte, key []byte) ([]byte, error) {
	tree := NewTree(db, true)
	err := tree.Load(roothash)
	if err != nil {
		return nil, err
	}
	_, proof, exit := tree.Proof(key)
	if exit {
		return proof, nil
	}
	return nil, nil
}

// DelKVPair 剔除key对应的节点在本次tree中，返回新的roothash和key对应的value
func DelKVPair(db dbm.DB, storeDel *types.StoreGet) ([]byte, [][]byte, error) {
	tree := NewTree(db, true)
	err := tree.Load(storeDel.StateHash)
	if err != nil {
		return nil, nil, err
	}
	values := make([][]byte, len(storeDel.Keys))
	for i := 0; i < len(storeDel.Keys); i++ {
		value, removed := tree.Remove(storeDel.Keys[i])
		if removed {
			values[i] = value
		}
	}
	return tree.Save(), values, nil
}

// VerifyKVPairProof 验证KVPair 的证明
func VerifyKVPairProof(db dbm.DB, roothash []byte, keyvalue types.KeyValue, proof []byte) bool {

	//通过传入的keyvalue构造leafnode
	leafNode := types.LeafNode{Key: keyvalue.GetKey(), Value: keyvalue.GetValue(), Height: 0, Size: 1}
	leafHash := leafNode.Hash()

	proofnode, err := ReadProof(roothash, leafHash, proof)
	if err != nil {
		treelog.Info("VerifyKVPairProof ReadProof err！", "err", err)
	}
	istrue := proofnode.Verify(keyvalue.GetKey(), keyvalue.GetValue(), roothash)
	if !istrue {
		treelog.Info("VerifyKVPairProof fail！", "keyBytes", keyvalue.GetKey(), "valueBytes", keyvalue.GetValue(), "roothash", roothash)
	}
	return istrue
}

// PrintTreeLeaf 通过roothash打印所有叶子节点
func PrintTreeLeaf(db dbm.DB, roothash []byte) {
	tree := NewTree(db, true)
	tree.Load(roothash)
	var i int32
	if tree.root != nil {
		leafs := tree.root.size
		treelog.Info("PrintTreeLeaf info")
		for i = 0; i < leafs; i++ {
			key, value := tree.GetByIndex(i)
			treelog.Info("leaf:", "index:", i, "key", string(key), "value", string(value))
		}
	}
}

// IterateRangeByStateHash 在start和end之间的键进行迭代回调[start, end)
func IterateRangeByStateHash(db dbm.DB, statehash, start, end []byte, ascending bool, fn func([]byte, []byte) bool) {
	tree := NewTree(db, true)
	tree.Load(statehash)
	//treelog.Debug("IterateRangeByStateHash", "statehash", hex.EncodeToString(statehash), "start", string(start), "end", string(end))

	tree.IterateRange(start, end, ascending, fn)
}

func genPrefixHashKey(node *Node, blockHeight int64) (key []byte) {
	//leafnode
	if node.height == 0 {
		key = []byte(fmt.Sprintf("%s-%010d-", leafNodePrefix, blockHeight))
	} else {
		key = []byte(fmt.Sprintf("%s-%010d-", hashNodePrefix, blockHeight))
	}
	return key
}
