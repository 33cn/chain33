package mavl

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/golang-lru"
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
	pruneHeight int = 10000
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
}

func isPruning() bool {
	return atomic.LoadInt32(&pruningState) == 1
}

func setPruning(state int32) {
	atomic.StoreInt32(&pruningState, state)
}

func pruningTree(db dbm.DB, curHeight int64) {
	setPruning(pruningStateStart)
	treelog.Info("pruningTree", "start curHeight:", curHeight)
	start := time.Now()
	pruningTreeLeafNode(db, curHeight)
	end := time.Now()
	treelog.Info("pruningTree", "curHeight:", curHeight, "pruning leafNode cost time:", end.Sub(start))
	setPruning(pruningStateEnd)
}

func pruningTreeLeafNode(db dbm.DB, curHeight int64) {
	prefix := []byte(leafKeyCountPrefix)
	it := db.Iterator(prefix, nil, true)
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
			}
			mp[string(key)] = append(mp[string(key)], data)
			count++
			if count >= onceScanCount {
				deleteNode(db, mp, curHeight, key)
				count = 0
			}
		}
	}
	if count > 0 {
		deleteNode(db, mp, curHeight, nil)
	}
}

func deleteNode(db dbm.DB, mp map[string][]hashData, curHeight int64, lastKey []byte) {
	if len(mp) == 0 {
		return
	}
	delMp := make(map[string]bool)
	batch := db.NewBatch(true)
	for key, vals := range mp {
		if len(vals) > 1 {
			for _, val := range vals[1:] { //从第二个开始判断
				if curHeight >= val.height+int64(pruneHeight) {
					//batch.Delete(val.hash) //叶子节点hash值的删除放入pruningHashNode中
					batch.Delete(genLeafCountKey([]byte(key), val.hash, val.height))
					delMp[string(val.hash)] = true
				}
			}
		}
	}
	batch.Write()
	if lastKey != nil {
		// 除了最后加入的key，删除已更新map key
		for key, _ := range mp {
			if !bytes.Equal([]byte(key), lastKey) {
				delete(mp, key)
			}
		}
	}
	//裁剪hashNode
	pruningHashNode(db, delMp)
}

func pruningHashNode(db dbm.DB, mp map[string]bool) {
	if len(mp) == 0 {
		return
	}
	ndb := newMarkNodeDB(db, 1024*10, mp)
	var strs []string
	count := 0
	for key, _ := range mp {
		mNode, err := ndb.LoadLeaf([]byte(key))
		if err == nil {
			strs = append(strs, mNode.getHash(ndb)...)
		}
		count++
	}
	//去重复
	mpN := make(map[string]bool)
	for _, str := range strs {
		mpN[str] = true
	}
	batch := db.NewBatch(true)
	count1 := 0
	for key, _  := range mpN {
		batch.Delete([]byte(key))
		count1++
	}
	fmt.Printf("pruningHashNode count %d , ndb.count %d delete %d \n", count, ndb.count, count1)
	treelog.Info("pruningHashNode ", "count ", count, "ndb.count", ndb.count, "delete node count", count1)
	batch.Write()
}

func (node *MarkNode) getHash(ndb *markNodeDB) (strs []string) {
	ndb.count++
	if node.hashPrune {
		strs = append(strs, string(node.hash))
	}
	//for test
	//hash := common.Bytes2Hex(node.hash[:2])
	//var b string
	//if len(node.brotherHash) > 3 {
	//	b = common.Bytes2Hex(node.brotherHash[:2])
	//}
	//var p string
	//if len(node.parentHash) > 3 {
	//	p = common.Bytes2Hex(node.parentHash[:2])
	//}
	//fmt.Printf("hash:%v left:%v right:%v\n", hash, b, p)
	broN := node.fetchBrotherNode(ndb)
	if broN == nil || (broN != nil && broN.hashPrune) {
		node.brotherPrune = true
		if node.hashPrune && node.brotherPrune {
			node.parentPrune = true
			//记录该节点需要裁剪
			ndb.addHash2Map(string(node.parentHash))
		}
		parN := node.fetchParentNode(ndb)
		if parN != nil {
			parN.hashPrune = node.parentPrune
			strs = append(strs, parN.getHash(ndb)...)
		}
	} else {
		// do nothing
	}
	return strs
}

func (ndb *markNodeDB)addHash2Map(str string) {
	ndb.mHash[str] = true
}

type MarkNode struct {
	height     int32
	hash       []byte
	hashPrune    bool
	parentHash   []byte
	parentNode   *MarkNode
	parentPrune  bool
	brotherNode  *MarkNode
	brotherHash  []byte
	brotherPrune bool
}

type markNodeDB struct {
	wg      sync.WaitGroup
	mtx     sync.Mutex
	cache   *lru.Cache //存储需要裁剪的节点
	db      dbm.DB
	mLeaf   map[string]bool
	mHash   map[string]bool
	count   int
}

func newMarkNodeDB(db dbm.DB, cache int, mp map[string]bool) *markNodeDB {
	cach, _ := lru.New(cache)
	ndb := &markNodeDB{
		cache: cach,
		db:    db,
		mLeaf:  mp,
		mHash: make(map[string]bool),
	}
	return ndb
}

func (ndb *markNodeDB) LoadLeaf(hash []byte) (node *MarkNode, err error) {
	if !bytes.Equal(hash, emptyRoot[:]) {
		leaf, err := ndb.fetchNode(hash)
		return leaf, err
	}
	return nil, types.ErrNotFound
}


func (node *MarkNode) fetchBrotherNode(ndb *markNodeDB) *MarkNode {
	if node.brotherNode != nil {
		return node.brotherNode
	} else {
		bNode, err := ndb.fetchNode(node.brotherHash)
		if err != nil {
			return nil
		}
		return bNode
	}
}

func (node *MarkNode) fetchParentNode(ndb *markNodeDB) *MarkNode {
	if node.parentNode != nil {
		return node.parentNode
	} else {
		pNode, err := ndb.fetchNode(node.parentHash)
		if err != nil {
			return nil
		}
		return pNode
	}
}

func (ndb *markNodeDB) fetchNode(hash []byte) (*MarkNode, error) {
	ndb.mtx.Lock()
	defer ndb.mtx.Unlock()

	var mNode *MarkNode
	// cache
	if ndb.cache != nil {
		elem, ok := ndb.cache.Get(string(hash))
		if ok {
			mNode = elem.(*MarkNode)
		}
	}
	if mNode == nil {
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
		mNode = &MarkNode{
			height:      node.height,
			hash:        node.hash,
			parentHash:  node.parentHash,
			brotherHash: node.brotherHash,
		}
		if ndb.cache != nil {
			ndb.cache.Add(string(hash), mNode)
		}
	}
	//检查是否删除
	if _, ok := ndb.mLeaf[string(hash)]; ok {
		mNode.hashPrune = true
	}
	if _, ok := ndb.mHash[string(hash)]; ok  {
		mNode.hashPrune = true
	}
	return mNode, nil
}

func pruningTreePrint(db dbm.DB, prefix []byte) {
	it := db.Iterator(prefix, nil, true)
	defer it.Close()
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		count++
	}
	fmt.Printf("prefix %s All count:%d \n", string(prefix), count)
	treelog.Info("pruningTree:", "prefix:", string(prefix), "All count", count)
}

//type pruneRootKv struct {
//	key   []byte
//	value *types.PruneRootNode
//}
//
//type funcProcess func(ndb *markNodeDB, pruneKv *pruneRootKv)
//
//func (ndb *markNodeDB) process(pruneKv *pruneRootKv, reqnum chan struct{}, cb funcProcess) {
//	beg := types.Now()
//	ndb.wg.Add(1)
//	defer func() {
//		<-reqnum
//		ndb.wg.Add(-1)
//		treelog.Debug("process deleteHashNode", "cost", types.Since(beg))
//	}()
//	cb(ndb, pruneKv)
//}

