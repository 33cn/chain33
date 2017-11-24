package mavl

import (
	"fmt"
	"sync"

	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
)

var treelog = log.New("module", "mavl")

//merkle avl tree
type MAVLTree struct {
	root *MAVLNode
	ndb  *nodeDB
}

// 新建一个merkle avl 树
func NewMAVLTree(db dbm.DB) *MAVLTree {
	if db == nil {
		// In-memory IAVLTree
		return &MAVLTree{}
	} else {
		// Persistent IAVLTree
		ndb := newNodeDB(db)
		return &MAVLTree{
			ndb: ndb,
		}
	}
}

func (t *MAVLTree) Copy() *MAVLTree {
	if t.root == nil {
		return &MAVLTree{
			root: nil,
			ndb:  t.ndb,
		}
	}
	if t.ndb != nil && !t.root.persisted {
		panic("It is unsafe to Copy() an unpersisted tree.")
	} else if t.ndb == nil && t.root.hash == nil {
		t.root.Hash(t)
	}
	return &MAVLTree{
		root: t.root,
		ndb:  t.ndb,
	}
}

// 获取tree的叶子节点数
func (t *MAVLTree) Size() int32 {
	if t.root == nil {
		return 0
	}
	return t.root.size
}

// 获取tree的高度
func (t *MAVLTree) Height() int32 {
	if t.root == nil {
		return 0
	}
	return t.root.height
}

//判断key是否存在tree中
func (t *MAVLTree) Has(key []byte) bool {
	if t.root == nil {
		return false
	}
	return t.root.has(t, key)
}

//设置k:v pair到tree中
func (t *MAVLTree) Set(key []byte, value []byte) (updated bool) {
	//treelog.Info("IAVLTree.Set", "key", key, "value",value)

	if t.root == nil {
		t.root = NewMAVLNode(key, value)
		return false
	}
	t.root, updated = t.root.set(t, key, value)
	return updated
}

//计算tree 的roothash
func (t *MAVLTree) Hash() []byte {
	if t.root == nil {
		return nil
	}
	hash := t.root.Hash(t)
	return hash
}

// 保存整个tree的节点信息到db中
func (t *MAVLTree) Save() []byte {
	if t.root == nil {
		return nil
	}
	if t.ndb != nil {
		t.root.save(t)
		t.ndb.Commit()
	}
	return t.root.hash
}

// 从db中加载rootnode
func (t *MAVLTree) Load(hash []byte) {
	if len(hash) == 0 {
		t.root = nil
	} else {
		t.root = t.ndb.GetNode(t, hash)
	}
}

//通过key获取leaf节点信息
func (t *MAVLTree) Get(key []byte) (index int32, value []byte, exists bool) {
	if t.root == nil {
		return 0, nil, false
	}
	return t.root.get(t, key)
}

//通过index获取leaf节点信息
func (t *MAVLTree) GetByIndex(index int32) (key []byte, value []byte) {
	if t.root == nil {
		return nil, nil
	}
	return t.root.getByIndex(t, index)
}

//获取指定k:v pair的proof证明
func (t *MAVLTree) Proof(key []byte) (value []byte, proofBytes []byte, exists bool) {
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

//-----------------------------------------------------------------------------

type nodeDB struct {
	mtx   sync.Mutex
	db    dbm.DB
	batch dbm.Batch
}

func newNodeDB(db dbm.DB) *nodeDB {
	ndb := &nodeDB{
		db:    db,
		batch: db.NewBatch(true),
	}
	return ndb
}

func (ndb *nodeDB) GetNode(t *MAVLTree, hash []byte) *MAVLNode {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	// Doesn't exist, load from db.
	buf := ndb.db.Get(hash)
	if len(buf) == 0 {
		panic(fmt.Sprintf("Value missing for key %X", hash))
	}
	node, err := MakeMAVLNode(buf, t)
	if err != nil {
		panic(fmt.Sprintf("Error reading IAVLNode. bytes: %X  error: %v", buf, err))
	}
	node.hash = hash
	node.persisted = true
	return node

}

func (ndb *nodeDB) SaveNode(t *MAVLTree, node *MAVLNode) {
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
	node.persisted = true
}

func (ndb *nodeDB) Commit() {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	// Write saves
	ndb.batch.Write()
	ndb.batch = ndb.db.NewBatch(true)
}

//对外接口
func SetKVPair(db dbm.DB, storeSet *types.StoreSet) []byte {
	tree := NewMAVLTree(db)
	tree.Load(storeSet.StateHash)

	for i := 0; i < len(storeSet.KV); i++ {
		tree.Set(storeSet.KV[i].Key, storeSet.KV[i].Value)
	}
	return tree.Save()
}

func GetKVPair(db dbm.DB, storeGet *types.StoreGet) [][]byte {
	tree := NewMAVLTree(db)
	tree.Load(storeGet.StateHash)
	var values [][]byte
	for i := 0; i < len(storeGet.Keys); i++ {
		_, value, exit := tree.Get(storeGet.Keys[i])
		if exit {
			values = append(values, value)
		}
	}
	return values
}

func GetKVPairProof(db dbm.DB, roothash []byte, key []byte) []byte {
	tree := NewMAVLTree(db)
	tree.Load(roothash)
	_, proof, exit := tree.Proof(key)
	if exit {
		return proof
	}
	return nil
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
