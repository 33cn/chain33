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

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/store/mavl/db/ticket"
	"github.com/33cn/chain33/types"
	farm "github.com/dgryski/go-farm"
	lru "github.com/hashicorp/golang-lru"
)

const (
	hashNodePrefix             = "_mh_"
	leafNodePrefix             = "_mb_"
	curMaxBlockHeight          = "_..mcmbh.._"
	rootHashHeightPrefix       = "_mrhp_"
	tkCloseCacheLen      int32 = 10 * 10000
)

var (
	// ErrNodeNotExist node is not exist
	ErrNodeNotExist = errors.New("ErrNodeNotExist")
	treelog         = log.New("module", "mavl")
	emptyRoot       [32]byte
	// 当前树的最大高度
	maxBlockHeight int64
	heightMtx      sync.Mutex
	memTree        MemTreeOpera
	tkCloseCache   MemTreeOpera
)

// InitGlobalMem 初始化全局变量
func InitGlobalMem(treeCfg *TreeConfig) {
	// 使能情况下非空创建当前整tree的缓存
	if treeCfg.EnableMemTree && memTree == nil {
		memTree = NewTreeMap(50 * 10000)
		if treeCfg.TkCloseCacheLen == 0 {
			treeCfg.TkCloseCacheLen = tkCloseCacheLen
		}
		tkCloseCache = NewTreeARC(int(treeCfg.TkCloseCacheLen))
	}
}

// ReleaseGlobalMem 释放全局缓存
func ReleaseGlobalMem() {
	if memTree != nil {
		memTree = nil
	}
	if tkCloseCache != nil {
		tkCloseCache = nil
	}
}

// TreeConfig ...
type TreeConfig struct {
	EnableMavlPrefix bool
	EnableMVCC       bool
	EnableMavlPrune  bool
	PruneHeight      int32
	EnableMemTree    bool
	EnableMemVal     bool
	TkCloseCacheLen  int32
}

type memNode struct {
	data   [][]byte //顺序为lefthash, righthash, key, value
	Height int32
	Size   int32
}

type uintkey uint64

// Tree merkle avl tree
type Tree struct {
	root        *Node
	ndb         *nodeDB
	blockHeight int64
	// 树更新之后，废弃的节点(更新缓存中节点先对旧节点进行删除然后再更新)
	obsoleteNode map[uintkey]struct{}
	updateNode   map[uintkey]*memNode
	tkCloseNode  map[uintkey]*memNode
	config       *TreeConfig
}

// NewTree 新建一个merkle avl 树
func NewTree(db dbm.DB, sync bool, treeCfg *TreeConfig) *Tree {
	if db == nil {
		// In-memory IAVLTree
		return &Tree{
			obsoleteNode: make(map[uintkey]struct{}),
			updateNode:   make(map[uintkey]*memNode),
			tkCloseNode:  make(map[uintkey]*memNode),
			config:       treeCfg,
		}
	}
	// Persistent IAVLTree
	ndb := newNodeDB(db, sync)
	return &Tree{
		ndb:          ndb,
		obsoleteNode: make(map[uintkey]struct{}),
		updateNode:   make(map[uintkey]*memNode),
		tkCloseNode:  make(map[uintkey]*memNode),
		config:       treeCfg,
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
		t.root = NewNode(copyBytes(key), copyBytes(value))
		return false
	}
	t.root, updated = t.root.set(t, copyBytes(key), copyBytes(value))
	return updated
}

func (t *Tree) getObsoleteNode() map[uintkey]struct{} {
	return t.obsoleteNode
}

// Hash 计算tree 的roothash
func (t *Tree) Hash() []byte {
	if t.root == nil {
		return nil
	}
	hash := t.root.Hash(t)
	// 更新memTree
	if t.config != nil && t.config.EnableMemTree && memTree != nil && tkCloseCache != nil {
		for k := range t.obsoleteNode {
			memTree.Delete(k)
		}
		for k, v := range t.updateNode {
			memTree.Add(k, v)
		}
		for k, v := range t.tkCloseNode {
			tkCloseCache.Add(k, v)
		}
		treelog.Debug("Tree.Hash", "memTree len", memTree.Len(), "tkCloseCache len", tkCloseCache.Len(), "tree height", t.blockHeight)
	}
	return hash
}

// Save 保存整个tree的节点信息到db中
func (t *Tree) Save() []byte {
	if t.root == nil {
		return nil
	}
	if t.ndb != nil {
		if t.config != nil && t.config.EnableMavlPrune && t.isRemoveLeafCountKey() {
			//DelLeafCountKV 需要先提前将leafcoutkey删除,这里需先于t.ndb.Commit()
			err := DelLeafCountKV(t.ndb.db, t.blockHeight, t.config)
			if err != nil {
				treelog.Error("Tree.Save", "DelLeafCountKV err", err)
			}
		}
		saveNodeNo := t.root.save(t)
		treelog.Debug("Tree.Save", "saveNodeNo", saveNodeNo, "tree height", t.blockHeight)
		// 保存每个高度的roothash
		if t.config != nil && t.config.EnableMavlPrune {
			err := t.root.saveRootHash(t)
			if err != nil {
				treelog.Error("Tree.Save", "saveRootHash err", err)
			}
		}

		beg := types.Now()
		err := t.ndb.Commit()
		treelog.Debug("tree.commit", "cost", types.Since(beg))
		if err != nil {
			return nil
		}
		// 该线程应只允许一个
		if t.config != nil && t.config.EnableMavlPrune && !isPruning() &&
			t.config.PruneHeight != 0 &&
			t.blockHeight%int64(t.config.PruneHeight) == 0 &&
			t.blockHeight/int64(t.config.PruneHeight) > 1 {
			wg.Add(1)
			go pruning(t.ndb.db, t.blockHeight, t.config)
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

// GetHash 通过key获取leaf节点hash信息
func (t *Tree) GetHash(key []byte) (index int32, hash []byte, exists bool) {
	if t.root == nil {
		return 0, nil, false
	}
	return t.root.getHash(t, key)
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
	proofBytes = types.Encode(&mavlproof)
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

func (t *Tree) getMaxBlockHeight() int64 {
	if t.ndb == nil || t.ndb.db == nil {
		return 0
	}
	value, err := t.ndb.db.Get([]byte(curMaxBlockHeight))
	if len(value) == 0 || err != nil {
		return 0
	}
	h := &types.Int64{}
	err = types.Decode(value, h)
	if err != nil {
		return 0
	}
	return h.Data
}

func (t *Tree) setMaxBlockHeight(height int64) error {
	if t.ndb == nil || t.ndb.batch == nil {
		return fmt.Errorf("ndb is nil")
	}
	h := &types.Int64{}
	h.Data = height
	t.ndb.batch.Set([]byte(curMaxBlockHeight), types.Encode(h))
	return nil
}

func (t *Tree) isRemoveLeafCountKey() bool {
	if t.ndb == nil || t.ndb.db == nil {
		return false
	}
	heightMtx.Lock()
	defer heightMtx.Unlock()

	if maxBlockHeight == 0 {
		maxBlockHeight = t.getMaxBlockHeight()
	}
	if t.blockHeight > maxBlockHeight {
		err := t.setMaxBlockHeight(t.blockHeight)
		if err != nil {
			panic(err)
		}
		maxBlockHeight = t.blockHeight
		return false
	}
	return true
}

// RemoveLeafCountKey 删除叶子节点的索引节点（防止裁剪时候回退产生的误删除）
func (t *Tree) RemoveLeafCountKey(height int64) {
	if t.root == nil || t.ndb == nil {
		return
	}
	prefix := genPrefixHashKey(&Node{}, height)
	it := t.ndb.db.Iterator(prefix, nil, true)
	defer it.Close()

	var keys [][]byte
	for it.Rewind(); it.Valid(); it.Next() {
		value := make([]byte, len(it.Value()))
		copy(value, it.Value())
		pData := &types.StoreNode{}
		err := types.Decode(value, pData)
		if err == nil {
			keys = append(keys, pData.Key)
		}
	}

	batch := t.ndb.db.NewBatch(true)
	for _, k := range keys {
		_, hash, exits := t.GetHash(k)
		if exits {
			batch.Delete(genLeafCountKey(k, hash, height, len(hash)))
			treelog.Debug("RemoveLeafCountKey:", "height", height, "key:", string(k), "hash:", common.ToHex(hash))
		}
	}
	dbm.MustWrite(batch)
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
	//从memtree中获取
	if t != nil && t.config != nil && t.config.EnableMemTree {
		node, err := getNodeMemTree(hash)
		if err == nil {
			return node, nil
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
	// Save node hashInt64 to memTree
	if t != nil && t.config != nil && t.config.EnableMemTree {
		updateGlobalMemTree(node, t.config)
	}
	return node, nil
}

// 获取叶子节点的所有父节点
func getHashNode(node *Node) (hashs [][]byte) {
	for {
		if node == nil {
			return
		}
		parN := node.parentNode
		if parN != nil {
			hashs = append(hashs, parN.hash)
			node = parN
		} else {
			break
		}
	}
	return hashs
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
	if t.config != nil && t.config.EnableMavlPrune && node.height == 0 {
		//save leafnode key&hash
		k := genLeafCountKey(node.key, node.hash, t.blockHeight, len(node.hash))
		data := &types.PruneData{
			Hashs: getHashNode(node),
		}
		ndb.batch.Set(k, types.Encode(data))
	}
	node.persisted = true
	ndb.cacheNode(node)
	delete(ndb.orphans, string(node.hash))
}

func getNodeMemTree(hash []byte) (*Node, error) {
	if memTree != nil {
		elem, ok := memTree.Get(uintkey(farm.Hash64(hash)))
		if ok {
			sn := elem.(*memNode)
			node := &Node{
				height:    sn.Height,
				size:      sn.Size,
				hash:      hash,
				persisted: true,
			}
			node.leftHash = sn.data[0]
			node.rightHash = sn.data[1]
			node.key = sn.data[2]
			if len(sn.data) == 4 {
				node.value = sn.data[3]
			}
			return node, nil
		}
	}

	if tkCloseCache != nil {
		// 从tkCloseCache缓存中获取
		elem, ok := tkCloseCache.Get(uintkey(farm.Hash64(hash)))
		if ok {
			sn := elem.(*memNode)
			node := &Node{
				height:    sn.Height,
				size:      sn.Size,
				hash:      hash,
				persisted: true,
			}
			node.leftHash = sn.data[0]
			node.rightHash = sn.data[1]
			node.key = sn.data[2]
			if len(sn.data) == 4 {
				node.value = sn.data[3]
			}
			return node, nil
		}
	}
	return nil, ErrNodeNotExist
}

// Save node hashInt64 to memTree
func updateGlobalMemTree(node *Node, treeCfg *TreeConfig) {
	if node == nil || memTree == nil || tkCloseCache == nil || treeCfg == nil {
		return
	}
	if !treeCfg.EnableMemVal && node.height == 0 {
		return
	}
	memN := &memNode{
		Height: node.height,
		Size:   node.size,
	}
	var isTkCloseNode bool
	if node.height == 0 {
		if bytes.HasPrefix(node.key, ticket.TicketPrefix) {
			tk := &ticket.Ticket{}
			err := types.Decode(node.value, tk)
			if err == nil && tk.Status == ticket.StatusCloseTicket { //ticket为close状态下不做存储
				isTkCloseNode = true
			}
		}
		memN.data = make([][]byte, 4)
		memN.data[3] = node.value
	} else {
		memN.data = make([][]byte, 3)
	}
	memN.data[0] = node.leftHash
	memN.data[1] = node.rightHash
	memN.data[2] = node.key
	if isTkCloseNode {
		tkCloseCache.Add(uintkey(farm.Hash64(node.hash)), memN)
	} else {
		memTree.Add(uintkey(farm.Hash64(node.hash)), memN)
	}
}

// Save node hashInt64 to localmem
func updateLocalMemTree(t *Tree, node *Node) {
	if t == nil || node == nil {
		return
	}
	if t.config != nil && !t.config.EnableMemVal && node.height == 0 {
		return
	}
	if t.updateNode != nil && t.tkCloseNode != nil {
		memN := &memNode{
			Height: node.height,
			Size:   node.size,
		}
		var isTkCloseNode bool
		if node.height == 0 {
			if bytes.HasPrefix(node.key, ticket.TicketPrefix) {
				tk := &ticket.Ticket{}
				err := types.Decode(node.value, tk)
				if err == nil && tk.Status == ticket.StatusCloseTicket { //ticket为close状态下不做存储
					isTkCloseNode = true
				}
			}
			memN.data = make([][]byte, 4)
			memN.data[3] = node.value
		} else {
			memN.data = make([][]byte, 3)
		}
		memN.data[0] = node.leftHash
		memN.data[1] = node.rightHash
		memN.data[2] = node.key
		if isTkCloseNode {
			t.tkCloseNode[uintkey(farm.Hash64(node.hash))] = memN
		} else {
			t.updateNode[uintkey(farm.Hash64(node.hash))] = memN
		}
		//treelog.Debug("Tree.SaveNode", "store struct size", unsafe.Sizeof(store), "byte size", len(storenode), "height", node.height)
	}
}

// cache缓存节点
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
	dbm.MustWrite(ndb.batch)

	ndb.batch = nil
	ndb.orphans = make(map[string]struct{})
	return nil
}

// SetKVPair 设置kv对外接口
func SetKVPair(db dbm.DB, storeSet *types.StoreSet, sync bool, treeCfg *TreeConfig) ([]byte, error) {
	tree := NewTree(db, sync, treeCfg)
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
func GetKVPair(db dbm.DB, storeGet *types.StoreGet, treeCfg *TreeConfig) ([][]byte, error) {
	tree := NewTree(db, true, treeCfg)
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
func GetKVPairProof(db dbm.DB, roothash []byte, key []byte, treeCfg *TreeConfig) ([]byte, error) {
	tree := NewTree(db, true, treeCfg)
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
func DelKVPair(db dbm.DB, storeDel *types.StoreGet, treeCfg *TreeConfig) ([]byte, [][]byte, error) {
	tree := NewTree(db, true, treeCfg)
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

// DelLeafCountKV 回退时候用于删除叶子节点的索引节点
func DelLeafCountKV(db dbm.DB, blockHeight int64, treeCfg *TreeConfig) error {
	treelog.Debug("RemoveLeafCountKey:", "height", blockHeight)
	prefix := genRootHashPrefix(blockHeight)
	it := db.Iterator(prefix, nil, true)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		hash, err := getRootHash(it.Key())
		if err == nil {
			tree := NewTree(db, true, treeCfg)
			err := tree.Load(hash)
			if err == nil {
				treelog.Debug("RemoveLeafCountKey:", "height", blockHeight, "root hash:", common.ToHex(hash))
				tree.RemoveLeafCountKey(blockHeight)
			}
		}
	}
	return nil
}

// VerifyKVPairProof 验证KVPair 的证明
func VerifyKVPairProof(db dbm.DB, roothash []byte, keyvalue *types.KeyValue, proof []byte) bool {

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
func PrintTreeLeaf(db dbm.DB, roothash []byte, treeCfg *TreeConfig) {
	tree := NewTree(db, true, treeCfg)
	err := tree.Load(roothash)
	if err != nil {
		return
	}
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
func IterateRangeByStateHash(db dbm.DB, statehash, start, end []byte, ascending bool, treeCfg *TreeConfig, fn func([]byte, []byte) bool) {
	tree := NewTree(db, true, treeCfg)
	err := tree.Load(statehash)
	if err != nil {
		return
	}
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

func genRootHashPrefix(blockHeight int64) (key []byte) {
	key = []byte(fmt.Sprintf("%s%010d", rootHashHeightPrefix, blockHeight))
	return key
}

func genRootHashHeight(blockHeight int64, hash []byte) (key []byte) {
	key = []byte(fmt.Sprintf("%s%010d%s", rootHashHeightPrefix, blockHeight, string(hash)))
	return key
}

func getRootHash(hashKey []byte) (hash []byte, err error) {
	if len(hashKey) < len(rootHashHeightPrefix)+blockHeightStrLen+sha256Len {
		return nil, types.ErrSize
	}
	if !bytes.Contains(hashKey, []byte(rootHashHeightPrefix)) {
		return nil, types.ErrSize
	}
	hash = hashKey[len(hashKey)-sha256Len:]
	return hash, nil
}
