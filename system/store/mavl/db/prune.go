package mavl

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"sync/atomic"
	"time"
)

const (
	rootNodePrefix     = "_mr_"
	leafKeyCountPrefix = "..mk.."
	blockHeightStrLen  = 10
	pruningStateStart  = 1
	pruningStateEnd    = 0
)

var (
	// 是否开启mavl裁剪
	enablePrune bool
	// 每个10000裁剪一次
	pruneHeight   int = 10000
	// 裁剪状态
	pruningState int32
)

func EnablePrune(enable bool) {
	enablePrune = enable
}

func SetPruneHeight(height int) {
	pruneHeight = height
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
	//for test
	//pruningTreePrint(db, []byte(leafKeyCountPrefix))
	//pruningTreePrint(db, []byte(hashNodePrefix))
	//pruningTreePrint(db, []byte(leafNodePrefix))

	treelog.Info("pruningTree", "start curHeight:", curHeight)
	start := time.Now()
	pruningTreeLeafNode(db, curHeight)
	end := time.Now()
	treelog.Info("pruningTree", "leafNode cost time:", end.Sub(start))
	//curHeight-prunBlockHeight 处遍历树，然后删除hashnode;缓存数节点到lru中
	pruningTreeHashNode(db, curHeight)
	end1 := time.Now()
	treelog.Info("pruningTree", "end curHeight:", curHeight, "hashNode cost time", end1.Sub(end))

	//for test
	//pruningTreePrint(db, []byte(leafKeyCountPrefix))
	//pruningTreePrint(db, []byte(hashNodePrefix))
	//pruningTreePrint(db, []byte(leafNodePrefix))
	setPruning(pruningStateEnd)
}

func pruningTreeLeafNode(db dbm.DB, curHeight int64) {
	prefix := []byte(leafKeyCountPrefix)
	it := db.Iterator(prefix, nil,true)
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

type mapStat struct {
	pruneLeafNum int32
	height       int64
}

func deleteNode(db dbm.DB, mp *map[string][]hashData, curHeight int64, lastKey []byte) {
	if mp == nil {
		return
	}
	// 存储是否需要裁剪hashnode的root数据
	upDateMp := make(map[string]mapStat)
	batch := db.NewBatch(true)
	for key, vals := range *mp {
		if len(vals) > 1 {
			for _, val := range vals[1:] { //从第二个开始判断
				if curHeight >= val.height+int64(pruneHeight) {
					batch.Delete(val.hash)
					batch.Delete(genLeafCountKey([]byte(key), val.hash, val.height))

					stat := upDateMp[string(genPrefixRootHash(val.rootHash, val.height))]
					stat.height = val.height
					stat.pruneLeafNum++
					upDateMp[string(genPrefixRootHash(val.rootHash, val.height))] = stat
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

func updatePruningState(db dbm.DB, mp *map[string]mapStat) {
	batch := db.NewBatch(true)
	for key, val := range *mp {
		value, err := db.Get([]byte(key))
		if err == nil && len(value) > 0 {
			root := types.PruneRootNode{}
			err = proto.Unmarshal(value, &root)
			if err == nil {
				root.NeedPruning = true
				root.PruneLeafNum += val.pruneLeafNum
				//treelog.Info("updatePruningState", "pruneLeafNum", root.PruneLeafNum)
				//fmt.Printf("updatePruningState pruneLeafNum %d \n", root.PruneLeafNum)
				v, err := proto.Marshal(&root)
				if err == nil {
					batch.Set([]byte(key), v)
				}
			}
		}
	}
	batch.Write()
}

type pruneRootKv struct {
	key   []byte
	value *types.PruneRootNode
}

func pruningTreeHashNode(db dbm.DB, curHeight int64) {
	pruneKvs := needPruningHashNodeTree(db, curHeight)
	if len(pruneKvs) == 0 {
		return
	}
	//协程池
	poolnum := make(chan struct{}, 10)
	ndb := newMarkNodeDB(db, 1024 * 100)
	for _, kv := range pruneKvs{
		poolnum <- struct{}{}
		go ndb.process(kv, poolnum, deleteHashNode)
	}
	ndb.wg.Wait()
}

func needPruningHashNodeTree(db dbm.DB, curHeight int64) (pruneKvs []*pruneRootKv) {
	prefix := []byte(rootNodePrefix)
	it := db.Iterator(prefix, nil, false)
	defer it.Close()

	hashlen := len(common.Hash{})
	for it.Rewind(); it.Valid(); it.Next() {
		value := it.Value()
		prRoot := &types.PruneRootNode{}
		err := proto.Unmarshal(value, prRoot)
		if err == nil {
			if curHeight < prRoot.Height+int64(pruneHeight) {
				//升序排列，可提前结束循环
				break
			}
			if prRoot.NeedPruning {
				hashK := make([]byte, len(it.Key()))
				copy(hashK, it.Key())
				if len(hashK) >= hashlen {
					pruneKvs = append(pruneKvs, &pruneRootKv{hashK, prRoot})
				}
			}
		}
	}
	return pruneKvs
}

type funcProcess func(ndb *markNodeDB, pruneKv *pruneRootKv)

func (ndb  *markNodeDB)process(pruneKv *pruneRootKv, reqnum chan struct{}, cb funcProcess) {
	beg := types.Now()
	ndb.wg.Add(1)
	defer func() {
		<-reqnum
		ndb.wg.Add(-1)
		treelog.Debug("process deleteHashNode", "cost", types.Since(beg))
	}()
	cb(ndb, pruneKv)
}

func deleteHashNode(ndb *markNodeDB, pruneKv *pruneRootKv) {
	if pruneKv == nil {
		return
	}
	hashlen := len(common.Hash{})
	t := NewMarkTree(ndb)
	if t == nil {
		return
	}
	hash := pruneKv.key[len(pruneKv.key)-hashlen:]
	err := t.Load(hash)
	if err != nil {
		return
	}
	strs := t.root.gethash(t)
	//fmt.Printf("---gethash loop count: %d ---\n", t.count)
	batch := ndb.db.NewBatch(true)
	for _, str := range strs {
		batch.Delete([]byte(str))
	}
	pruneKv.value.NeedPruning = false
	v, _ := proto.Marshal(pruneKv.value)
	batch.Set(pruneKv.key, v)
	batch.Write()
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

type markNodeDB struct {
	wg      sync.WaitGroup
	mtx     sync.Mutex
	cache   *lru.Cache
	db      dbm.DB
}

type MarkTree struct {
	root *MarkNode
	ndb  *markNodeDB
	count int
}

func newMarkNodeDB(db dbm.DB, cache int) *markNodeDB {
	cach, _ := lru.New(cache)
	ndb := &markNodeDB{
		cache:   cach,
		db:      db,
	}
	return ndb
}

func NewMarkTree(ndb *markNodeDB) *MarkTree {
	if ndb == nil {
		// In-memory IAVLTree
		return nil
	} else {
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

func (ndb *markNodeDB) fetchNode(t *MarkTree, hash []byte) (*MarkNode, error) {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	// Check the cache.
	if ndb.cache != nil {
		elem, ok := ndb.cache.Get(string(hash))
		if ok {
			//treelog.Info("************for tree test************")
			return elem.(*MarkNode), nil
		}
	}

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
	if ndb.cache != nil {
		ndb.cache.Add(string(hash), mNode)
	}
	return mNode, nil
}

func pruningTreePrint(db dbm.DB, prefix []byte) {
	it := db.Iterator(prefix, nil, true)
	defer it.Close()
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		//copy
		//if bytes.Equal(prefix, []byte(leafKeyCountPrefix)) {
		//	hashK := make([]byte, len(it.Key()))
		//	copy(hashK, it.Key())
		//
		//	value := it.Value()
		//	var pData types.PruneData
		//	err := proto.Unmarshal(value, &pData)
		//	if err != nil {
		//		panic("Unmarshal mavl leafCountKey fail")
		//	}
		//	hashLen := int(pData.Lenth)
		//	_, err = getKeyFromLeafCountKey(hashK, hashLen)
		//	if err == nil {
		//		//fmt.Printf("key:%s height:%d \n", string(key), pData.Height)
		//	}
		//}
		count++
	}
	fmt.Printf("prefix %s All count:%d \n", string(prefix), count)
	treelog.Info("pruningTree:", "prefix:", string(prefix), "All count", count)
}
