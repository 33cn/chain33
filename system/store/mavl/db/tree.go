package mavl

import (
	"bytes"
	"errors"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"sort"
	"sync/atomic"
)

const (
	hashNodePrefix     = "_mh_"
	leafNodePrefix     = "_mb_"
	rootNodePrefix     = "_mr_"
	leafKeyCountPrefix = "..mk.."
	blockHeightStrLen  = 10
	pruningStateStart  = 1
	pruningStateEnd    = 0
)

var (
	ErrNodeNotExist  = errors.New("ErrNodeNotExist")
	treelog          = log.New("module", "mavl")
	emptyRoot        [32]byte
	enableMavlPrefix bool
	// 是否开启MVCC
	enableMvcc bool
	// 是否开启mavl裁剪
	enablePrune bool
	// 每个100000裁剪一次
	pruneBlockHeight int = 100000
	// 裁剪状态
	pruningState int32
)

func EnableMavlPrefix(enable bool) {
	enableMavlPrefix = enable
}

func EnableMVCC(enable bool) {
	enableMvcc = enable
}

func EnablePrune(enable bool) {
	enablePrune = enable
}

func SetPrunBlockHeight(height int) {
	pruneBlockHeight = height
}

//merkle avl tree
type Tree struct {
	root *Node
	ndb  *nodeDB
	//batch *nodeBatch
	//randomstr string
	blockHeight int64
}

// 新建一个merkle avl 树
func NewTree(db dbm.DB, sync bool) *Tree {
	if db == nil {
		// In-memory IAVLTree
		return &Tree{
		//randomstr: common.GetRandString(5),
		}
	} else {
		// Persistent IAVLTree
		ndb := newNodeDB(db, sync)
		return &Tree{
			ndb: ndb,
			//randomstr: common.GetRandString(5),
		}
	}
}

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

// 获取tree的叶子节点数
func (t *Tree) Size() int32 {
	if t.root == nil {
		return 0
	}
	return t.root.size
}

// 获取tree的高度
func (t *Tree) Height() int32 {
	if t.root == nil {
		return 0
	}
	return t.root.height
}

//判断key是否存在tree中
func (t *Tree) Has(key []byte) bool {
	if t.root == nil {
		return false
	}
	return t.root.has(t, key)
}

//设置k:v pair到tree中
func (t *Tree) Set(key []byte, value []byte) (updated bool) {
	//treelog.Info("IAVLTree.Set", "key", key, "value",value)
	if t.root == nil {
		t.root = NewNode(key, value)
		return false
	}
	t.root, updated = t.root.set(t, key, value)
	return updated
}

//计算tree 的roothash
func (t *Tree) Hash() []byte {
	if t.root == nil {
		return nil
	}
	hash := t.root.Hash(t)
	return hash
}

// 保存整个tree的节点信息到db中
func (t *Tree) Save() []byte {
	if t.root == nil {
		return nil
	}
	if t.ndb != nil {
		saveNodeNo := t.root.save(t)
		treelog.Debug("Tree.Save", "saveNodeNo", saveNodeNo)
		err := t.ndb.Commit()
		if err != nil {
			return nil
		}
		// 该线程应只允许一个
		//if enablePrune && !isPruning() && t.blockHeight%int64(pruneBlockHeight) == 0 {
		//	go pruningTree(t.ndb.db, t.blockHeight)
		//}
	}
	return t.root.hash
}

// 从db中加载rootnode
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

func (t *Tree) SetBlockHeight(height int64) {
	t.blockHeight = height
}

//通过key获取leaf节点信息
func (t *Tree) Get(key []byte) (index int32, value []byte, exists bool) {
	if t.root == nil {
		return 0, nil, false
	}
	return t.root.get(t, key)
}

//通过index获取leaf节点信息
func (t *Tree) GetByIndex(index int32) (key []byte, value []byte) {
	if t.root == nil {
		return nil, nil
	}
	return t.root.getByIndex(t, index)
}

//获取指定k:v pair的proof证明
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

//删除key对应的节点
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

// 依次迭代遍历树的所有键
func (t *Tree) Iterate(fn func(key []byte, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverse(t, true, func(node *Node) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		} else {
			return false
		}
	})
}

// 在start和end之间的键进行迭代回调[start, end)
func (t *Tree) IterateRange(start, end []byte, ascending bool, fn func(key []byte, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverseInRange(t, start, end, ascending, false, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		} else {
			return false
		}
	})
}

// 在start和end之间的键进行迭代回调[start, end]
func (t *Tree) IterateRangeInclusive(start, end []byte, ascending bool, fn func(key, value []byte) bool) (stopped bool) {
	if t.root == nil {
		return false
	}
	return t.root.traverseInRange(t, start, end, ascending, true, 0, func(node *Node, _ uint8) bool {
		if node.height == 0 {
			return fn(node.key, node.value)
		} else {
			return false
		}
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

func (ndb *nodeDB) GetBatch(sync bool) *nodeBatch {
	return &nodeBatch{ndb.db.NewBatch(sync)}
}

//保存节点
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
			RHash:  t.root.hash,
		}
		v, err := proto.Marshal(data)
		if err != nil {
			panic(err)
		}
		ndb.batch.Set(k, v)
	} else if enablePrune && node.height == t.root.height {
		// save prefix root
		k := genPrefixRootHash(node.hash, t.blockHeight)
		pruneNode := &types.PruneRootNode{
			NeedPruning: false,
			Height:      t.blockHeight,
			LeftHash:    node.leftHash,
			RightHash:   node.rightHash,
		}
		v, err := proto.Marshal(pruneNode)
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

//删除节点
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

//提交状态tree，批量写入db中
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

//对外接口
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

//剔除key对应的节点在本次tree中，返回新的roothash和key对应的value
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

func genPrefixRootHash(hash []byte, blockHeight int64) (key []byte) {
	key = []byte(fmt.Sprintf("%s-%010d-%s", rootNodePrefix, blockHeight, string(hash)))
	return key
}

func genLeafCountKey(key, hash []byte, height int64) (hashkey []byte) {
	hashkey = []byte(fmt.Sprintf("%s%s%010d%s", leafKeyCountPrefix, string(key), height, string(hash)))
	return hashkey
}

func getKeyFromLeafCountKey(hashkey []byte, hashlen int) ([]byte, error) {
	if len(hashkey) <= len(leafKeyCountPrefix)+hashlen+blockHeightStrLen {
		return nil, types.ErrSize
	}
	if !bytes.Contains(hashkey, []byte(leafKeyCountPrefix)) {
		return nil, types.ErrSize
	}
	k := bytes.TrimPrefix(hashkey, []byte(leafKeyCountPrefix))
	k = k[:len(k)-hashlen-blockHeightStrLen]
	return k, nil
}

type hashData struct {
	height   int64
	hash     []byte
	rootHash []byte
}

func isPruning() bool {
	return atomic.LoadInt32(&pruningState) == 1
}

func setPruning(state int32) {
	atomic.StoreInt32(&pruningState, state)
}

func pruningTree(db dbm.DB, curHeight int64) {
	setPruning(pruningStateStart)
	pruningTreeLeafNode(db, curHeight)
	//TODO curHeight-prunBlockHeight 处遍历树，然后删除hashnode;缓存数节点到lru中
	pruningTreeHashNode(db, curHeight)
	setPruning(pruningStateEnd)
}

func pruningTreeLeafNode(db dbm.DB, curHeight int64) {
	prefix := []byte(leafKeyCountPrefix)
	it := db.Iterator(prefix, true)
	defer it.Close()

	const onceScanCount = 100000
	mp := make(map[string][]hashData)
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		//copy key
		hashK := make([]byte, len(it.Key()))
		copy(hashK, it.Key())

		value := it.Value()
		var pData types.PruneData
		err := proto.Unmarshal(value, &pData)
		if err != nil {
			panic("Unmarshal mavl leafCountKey fail")
		}
		hashLen := int(pData.Lenth)
		key, err := getKeyFromLeafCountKey(hashK, hashLen)
		if err == nil {
			data := hashData{
				height:   pData.Height,
				hash:     hashK[len(hashK)-hashLen:],
				rootHash: pData.RHash,
			}
			mp[string(key)] = append(mp[string(key)], data)
			count++
			if count >= onceScanCount {
				deleteNode(db, &mp, curHeight, key)
				count = 0
			}
		}
	}
	deleteNode(db, &mp, curHeight, nil)
}

func deleteNode(db dbm.DB, mp *map[string][]hashData, curHeight int64, lastKey []byte) {
	if mp == nil {
		return
	}
	// 存储是否需要裁剪hashnode的root数据
	upDateMp := make(map[string]int64)
	batch := db.NewBatch(true)
	for key, vals := range *mp {
		if len(vals) > 1 {
			for _, val := range vals[1:] { //从第二个开始判断
				if curHeight >= val.height+int64(pruneBlockHeight) {
					batch.Delete(val.hash)
					batch.Delete(genLeafCountKey([]byte(key), val.hash, val.height))
					upDateMp[string(val.rootHash)] = val.height
				}
			}
		}
	}
	batch.Write()
	if lastKey != nil {
		// 除了最后加入的key，删除已更新map key
		for key, _ := range *mp {
			if !bytes.Equal([]byte(key), lastKey) {
				delete(*mp, key)
			}
		}
	}
	// 更新裁剪状态
	updatePruningState(db, &upDateMp)
}

func updatePruningState(db dbm.DB, mp *map[string]int64) {
	var strs []string
	for key, val := range *mp {
		strs = append(strs, string(genPrefixRootHash([]byte(key), val)))
	}
	sort.Strings(strs)
	batch := db.NewBatch(true)
	for _, str := range strs {
		value, err := db.Get([]byte(str))
		if err == nil && len(value) > 0 {
			root := types.PruneRootNode{}
			err = proto.Unmarshal(value, &root)
			if err == nil {
				root.NeedPruning = true
				val, err := proto.Marshal(&root)
				if err == nil {
					batch.Set([]byte(str), val)
				}
			}
		}
	}
	batch.Write()
}

func pruningTreeHashNode(db dbm.DB, curHeight int64) {
	prefix := []byte(rootNodePrefix)
	it := db.Iterator(prefix, false)
	defer it.Close()

	hashlen := len(common.Hash{})
	for it.Rewind(); it.Valid(); it.Next() {
		value := it.Value()
		prRoot := &types.PruneRootNode{}
		err := proto.Unmarshal(value, prRoot)
		if err == nil {
			if curHeight < prRoot.Height+int64(pruneBlockHeight) {
				//升序排列，可提前结束循环
				break
			}
			if prRoot.NeedPruning  {
				hashK := make([]byte, len(it.Key()))
				copy(hashK, it.Key())
				if len(hashK) >= hashlen {
					tree := NewMarkTree(db, true)
					hash := hashK[len(hashK)-hashlen:]
					err := tree.Load(hash)
					if err == nil {
						batch := db.NewBatch(true)
						deleteHashNode(tree, batch)
						// 将rootkey 置为修改完成
						prRoot.NeedPruning = false
						v, _ := proto.Marshal(prRoot)
						batch.Set(hashK, v)
						batch.Write()
					}
				}
			}
		}
	}
}

func deleteHashNode(t *MarkTree, batch dbm.Batch) {
	if t == nil {
		return
	}
	strs := t.root.gethash(t)
	//fmt.Printf("---gethash loop count: %d ---\n", t.count)
	for _, str := range strs {
		batch.Delete([]byte(str))
	}
}

type MarkNode struct {
	height     int32
	hash       []byte
	leftHash   []byte
	leftNode   *MarkNode
	leftPurne  bool
	rightHash  []byte
	rightNode  *MarkNode
	rightPurne bool
}
type MarkTree struct {
	root *MarkNode
	ndb  *nodeDB
	count int
}

func NewMarkTree(db dbm.DB, sync bool) *MarkTree {
	if db == nil {
		// In-memory IAVLTree
		return &MarkTree{
		}
	} else {
		// Persistent IAVLTree
		ndb := newNodeDB(db, sync)
		return &MarkTree{
			ndb: ndb,
		}
	}
}

func (t *MarkTree) Load(hash []byte) (err error) {
	if hash == nil {
		return
	}
	if !bytes.Equal(hash, emptyRoot[:]) {
		t.root, err = t.ndb.fetchNode(t, hash)
		return err
	}
	return nil
}

func (node *MarkNode) gethash(t *MarkTree) (strs []string) {
	t.count++
	if node.height == 0 {
		//do nothing
	} else {
		//hash := common.Bytes2Hex(node.hash[:2])
		//left := common.Bytes2Hex(node.leftHash[:2])
		//right := common.Bytes2Hex(node.rightHash[:2])
		//fmt.Printf("hash:%v left:%v right:%v\n", hash, left, right)
		leftN := node.fetchLeftNode(t)
		if leftN != nil {
			strs = append(strs, leftN.gethash(t)...)
		}
		if leftN == nil {
			node.leftPurne = true
		}else if leftN.leftPurne && leftN.rightPurne {
			node.leftPurne = true
		}
		//
		rightN := node.fetchRightNode(t)
		if rightN != nil {
			strs = append(strs, rightN.gethash(t)...)
		}
		if rightN == nil {
			node.rightPurne = true
		} else if rightN.leftPurne && rightN.rightPurne{
			node.rightPurne = true
		}
		//
		if node.leftPurne && node.rightPurne {
			strs = append(strs, string(node.hash))
		}
	}
	return strs
}

func (node *MarkNode) fetchLeftNode(t *MarkTree) *MarkNode {
	if node.leftNode != nil {
		return node.leftNode
	} else {
		leftNode, err := t.ndb.fetchNode(t, node.leftHash)
		if err != nil {
			return nil
		}
		return leftNode
	}
}

func (node *MarkNode) fetchRightNode(t *MarkTree) *MarkNode {
	if node.rightNode != nil {
		return node.rightNode
	} else {
		rightNode, err := t.ndb.fetchNode(t, node.rightHash)
		if err != nil {
			return nil
		}
		return rightNode
	}
}

//区别主要是对于leaf节点这不缓存
func (ndb *nodeDB) fetchNode(t *MarkTree, hash []byte) (*MarkNode, error) {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	// Check the cache.

	var buf []byte
	buf, err := ndb.db.Get(hash)

	if len(buf) == 0 || err != nil {
		return nil, ErrNodeNotExist
	}
	node, err := MakeNode(buf, nil)
	if err != nil {
		panic(fmt.Sprintf("Error reading IAVLNode. bytes: %X  error: %v", buf, err))
	}
	node.hash = hash
	mNode := &MarkNode{
		height: node.height,
		hash: node.hash,
		leftHash: node.leftHash,
		rightHash: node.rightHash,
	}
	//后续加
	//ndb.cacheNode(node)
	return mNode, nil
}

func pruningTreePrint(db dbm.DB, prefix []byte) {
	it := db.Iterator(prefix, true)
	defer it.Close()
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		//copy
		if bytes.Equal(prefix, []byte(leafKeyCountPrefix)) {
			hashK := make([]byte, len(it.Key()))
			copy(hashK, it.Key())

			value := it.Value()
			var pData types.PruneData
			err := proto.Unmarshal(value, &pData)
			if err != nil {
				panic("Unmarshal mavl leafCountKey fail")
			}
			hashLen := int(pData.Lenth)
			_, err = getKeyFromLeafCountKey(hashK, hashLen)
			if err == nil {
				//fmt.Printf("key:%s height:%d \n", string(key), pData.Height)
			}
		}
		count++
	}
	fmt.Printf("prefix %s All count:%d \n", string(prefix), count)
}
