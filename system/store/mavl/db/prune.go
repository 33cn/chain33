package mavl

import (
	"bytes"
	"fmt"
	"sync"

	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/common"
)

const (
	rootNodePrefix     = "_mr_"
	leafKeyCountPrefix = "..mk.."
	delMapPoolPrefix   = "_..md.._"
	blockHeightStrLen  = 10
	pruningStateStart  = 1
	pruningStateEnd    = 0
	delNodeCacheSize   = 1024 * 10
)

var (
	// 是否开启mavl裁剪
	enablePrune  bool
	// 每个10000裁剪一次
	pruneHeight  int = 10000
	// 裁剪状态
	pruningState int32
	delMapPoolCache *lru.Cache
)

func init() {
	cache, err := lru.New(delNodeCacheSize)
	if err != nil {
		panic(fmt.Sprint("new delNodeCache lru fail", err))
	}
	delMapPoolCache = cache
}

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

func genDeletePoolKey(hash []byte) (key, value []byte) {
	if len(hash) < 32 {
		panic("genDeletePoolKey error hash len illegal")
	}
	hashLen := len(common.Hash{})
	if len(hash) > hashLen {
		value = hash[len(hash) - hashLen:]
	} else {
		value = hash
	}
	key = value[:1]
	key = []byte(fmt.Sprintf("%s%s", delMapPoolPrefix, string(key)))
	return key, value
}

type hashData struct {
	height int64
	hash   []byte
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
				height: pData.Height,
				hash:   hashK[len(hashK)-hashLen:],
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
		for key := range mp {
			if !bytes.Equal([]byte(key), lastKey) {
				delete(mp, key)
			}
		}
	}
	//裁剪hashNode
	pruningHashNode(db, delMp)
}

type addHash struct {
	hash  []byte
	isAdd bool
}

func pruningHashNode(db dbm.DB, mp map[string]bool) {
	if len(mp) == 0 {
		return
	}
	ndb := newMarkNodeDB(db, 1024 * 10, mp)
	var addMpStrs   []string
	var delMpStrs   []string
	var delNodeStrs []string
	count := 0
	for key := range mp {
		mNode, err := ndb.LoadLeaf([]byte(key))
		if err == nil {
			add, del, delN := mNode.gethash(ndb)
			addMpStrs = append(addMpStrs, add...)
			delMpStrs = append(delMpStrs, del...)
			delNodeStrs = append(delNodeStrs, delN...)
		}
		count++
	}

	//根据keyMap进行归类
	mpAddDel := make(map[string][]addHash)
	for _, str := range addMpStrs  {
		key, hash := genDeletePoolKey([]byte(str))
		//fmt.Println("addMpStrs", common.ToHex(hash[:2]))
		data := addHash{
			hash:  hash,
			isAdd: true,
		}
		mpAddDel[string(key)] = append(mpAddDel[string(key)], data)
	}

	for _, str := range delMpStrs  {
		key, hash := genDeletePoolKey([]byte(str))
		//fmt.Println("delMpStrs", common.ToHex(hash[:2]))
		data := addHash{
			hash:  hash,
			isAdd: false,
		}
		mpAddDel[string(key)] = append(mpAddDel[string(key)], data)
	}

	//更新map集数据
	batch := db.NewBatch(true)
	for mpk, mpV := range mpAddDel {
		delMp := ndb.getMapPool(mpk)
		if delMp != nil {
			for _, aHsh := range mpV {
				if aHsh.isAdd {
					delMp.MpNode[string(aHsh.hash)] = true
				} else {
					delete(delMp.MpNode, string(aHsh.hash))
				}
			}
			ndb.updateHash2Map(batch, mpk, delMp)
		}
	}
	batch.Write()

	//将叶子节点删除
	mpN := make(map[string]bool)
	for _, str := range delNodeStrs {
		mpN[str] = true
	}
	batch = db.NewBatch(true)
	count1 := 0
	for key := range mpN {
		//fmt.Println("delNodeStrs", common.ToHex([]byte(key)[:2]))
		batch.Delete([]byte(key))
		count1++
	}
	batch.Write()
	//fmt.Printf("pruningHashNode count %d , ndb.count %d delete %d \n", count, ndb.count, count1)
	treelog.Info("pruningHashNode ", "count ", count, "ndb.count", ndb.count, "delete node count", count1)
}

func (node *MarkNode) gethash(ndb *markNodeDB) (addMpStrs, delMpStrs, delNodeStrs []string) {
	//目前只裁剪第0,1层节点，暂时不使用递归
	parN := node.fetchParentNode(ndb)
	if parN != nil {
		if bytes.Equal(parN.leftHash, node.hash) {
			node.brotherHash = parN.rightHash
		} else {
			node.brotherHash = parN.leftHash
		}
		/*******************/
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
		/*******************/

		broN := node.fetchBrotherNode(ndb)
		if broN == nil || (broN != nil && broN.hashPrune) {
			node.parentPrune = true
		}
		if node.parentPrune {
			if broN == nil {
				//将兄弟节点从已删除map部分移除
				delMpStrs = append(delMpStrs, string(node.brotherHash))
			}
			//将父节点删除
			delNodeStrs = append(delNodeStrs, string(node.parentHash))
		} else {
			if parN.height == 1 {
				// 需要将本节点加入的已删除map部分,只针对父节点为1,
				// 父节点大于1不处理当前只裁剪叶子节点之上一层
				addMpStrs = append(addMpStrs, string(node.hash))
			}
		}
		//将当前节点删除
		delNodeStrs = append(delNodeStrs, string(node.hash))
	}
	return addMpStrs, delMpStrs, delNodeStrs
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

func (ndb *markNodeDB) updateHash2Map(batch dbm.Batch, key string, mp *types.DeleteNodeMap) {
	if ndb.delMpCache != nil {
		ndb.delMpCache.Add(key, mp)
	}
	v, err := proto.Marshal(mp)
	if err != nil {
		panic(fmt.Sprint("types.DeleteNodeMap fail", err))
	}
	batch.Set([]byte(key), v)
}

// str为mappool的key即genDeletePoolKey产生
func (ndb *markNodeDB) getMapPool(str string) (mp *types.DeleteNodeMap) {
	if ndb.delMpCache != nil {
		elem, ok := ndb.delMpCache.Get(string(str))
		if ok {
			mp = elem.(*types.DeleteNodeMap)
			return mp
		}
	}
	v, err := ndb.db.Get([]byte(str))
	if err != nil || len(v) == 0 {
		//如果不存在说明是新的则
		//创建一个空的map集
		m := types.DeleteNodeMap{}
		m.MpNode = make(map[string]bool)
		return &m
	}
	m := types.DeleteNodeMap{}
	err = proto.Unmarshal(v, &m)
	if err != nil {
		return nil
	}
	if ndb.delMpCache != nil {
		if ndb.delMpCache.Len() > delNodeCacheSize {
			strs := ndb.delMpCache.Keys()
			elem, ok := ndb.delMpCache.Get(strs[0])
			if ok {
				mp = elem.(*types.DeleteNodeMap)
				v, err := proto.Marshal(mp)
				if err != nil {
					panic(fmt.Sprint("types.DeleteNodeMap fail", err))
				}
				ndb.db.Set([]byte(strs[0].(string)), v)
			}
		}
		ndb.delMpCache.Add(string(str), m)
	}
	return &m
}

func (ndb *markNodeDB) addHash2Map(str string) {
	ndb.mHash[str] = true
}

type MarkNode struct {
	height       int32
	hash         []byte
	hashPrune    bool
	leftHash     []byte
	rightHash    []byte
	parentHash   []byte
	parentNode   *MarkNode
	parentPrune  bool
	brotherNode  *MarkNode
	brotherHash  []byte
	brotherPrune bool
}

type markNodeDB struct {
	wg    sync.WaitGroup
	mtx   sync.Mutex
	cache      *lru.Cache  // 缓存从数据库中读取的叶子节点
	delMpCache *lru.Cache  // 缓存从数据库中的map
	db    dbm.DB
	mLeaf map[string]bool
	mHash map[string]bool
	count int
}

func newMarkNodeDB(db dbm.DB, cache int, mp map[string]bool) *markNodeDB {
	cach, _ := lru.New(cache)
	ndb := &markNodeDB{
		cache: cach,
		delMpCache: delMapPoolCache,
		db:    db,
		mLeaf: mp,
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
		// 先判断是否已经删除掉,如果删除掉查找比较耗时
		key, hsh := genDeletePoolKey(hash)
		mp := ndb.getMapPool(string(key))
		if mp != nil {
			if _, ok := mp.MpNode[string(hsh)]; ok {
				return nil, ErrNodeNotExist
			}
		}

		var buf []byte
		buf, err := ndb.db.Get(hash)
		if len(buf) == 0 || err != nil {
			treelog.Info("fetchNode this not happend, because have map pool:", "err:", err)
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
			leftHash:    node.leftHash,
			rightHash:   node.rightHash,
			parentHash:  node.parentHash,
		}
		if ndb.cache != nil {
			ndb.cache.Add(string(hash), mNode)
		}
	}
	//检查是否删除
	if _, ok := ndb.mLeaf[string(hash)]; ok {
		mNode.hashPrune = true
	}
	//if _, ok := ndb.mHash[string(hash)]; ok {
	//	mNode.hashPrune = true
	//}
	return mNode, nil
}

func PruningTreePrint(db dbm.DB, prefix []byte) {
	it := db.Iterator(prefix, nil, true)
	defer it.Close()
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		if bytes.Equal(prefix, []byte(leafKeyCountPrefix)) {
			hashK := it.Key()
			value := it.Value()
			var pData types.PruneData
			err := proto.Unmarshal(value, &pData)
			if err == nil {
				hashLen := int(pData.Lenth)
				key, err := getKeyFromLeafCountKey(hashK, hashLen)
				if err == nil {
					treelog.Info("pruningTree:", "key:", string(key), "height", pData.Height)
				}
			}
		} else if bytes.Equal(prefix, []byte(hashNodePrefix)) {
			treelog.Info("pruningTree:", "key:", string(it.Key()))
		} else if bytes.Equal(prefix, []byte(leafNodePrefix)) {
			treelog.Info("pruningTree:", "key:", string(it.Key()))
		} else if bytes.Equal(prefix, []byte(delMapPoolPrefix)) {
			value := it.Value()
			var pData types.DeleteNodeMap
			err := proto.Unmarshal(value, &pData)
			if err == nil {
				for k, _ := range pData.MpNode {
					treelog.Info("delMapPool value ", "hash:", common.Bytes2Hex([]byte(k)[:2]) )
				}
			}
		}
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
