package mavl

import (
	"bytes"
	"fmt"
	"sync"

	"sync/atomic"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/hashicorp/golang-lru"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	leafKeyCountPrefix = "..mk.."
	delMapPoolPrefix   = "_..md.._"
	blockHeightStrLen  = 10
	pruningStateStart  = 1
	pruningStateEnd    = 0
	//删除节点pool以hash的首字母为key因此有256个
	delNodeCacheSize = 256 + 1
	//每个del Pool下存放默认4096个hash
	perDelNodePoolSize = 4096
)

var (
	// 是否开启mavl裁剪
	enablePrune bool
	// 每个10000裁剪一次
	pruneHeight int = 10000
	// 裁剪状态
	pruningState int32
	delPoolCache *lru.Cache
	wg           sync.WaitGroup
	quit         bool
)

func init() {
	cache, err := lru.New(delNodeCacheSize)
	if err != nil {
		panic(fmt.Sprint("new delNodeCache lru fail", err))
	}
	delPoolCache = cache
}

type delNodeValuePool struct {
	delCache *lru.Cache
}

type hashData struct {
	height int64
	hash   []byte
}

type addHash struct {
	hash  []byte
	isAdd bool
}

type addDelStr struct {
	isAdd bool
	str   string
}

func NewDelNodeValuePool(cacSize int) *delNodeValuePool {
	cache, err := lru.New(cacSize)
	if err != nil {
		return nil
	}
	dNodePool := &delNodeValuePool{}
	dNodePool.delCache = cache
	return dNodePool
}

func EnablePrune(enable bool) {
	enablePrune = enable
}

func SetPruneHeight(height int) {
	pruneHeight = height
}

func ClosePrune() {
	quit = true
	wg.Wait()
	//防止BaseStore没有关闭再次进入
	setPruning(pruningStateStart)
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
		value = hash[len(hash)-hashLen:]
	} else {
		value = hash
	}
	key = value[:1]
	key = []byte(fmt.Sprintf("%s%s", delMapPoolPrefix, string(key)))
	return key, value
}

func isPruning() bool {
	return atomic.LoadInt32(&pruningState) == 1
}

func setPruning(state int32) {
	atomic.StoreInt32(&pruningState, state)
}

func pruningTree(db dbm.DB, curHeight int64) {
	wg.Add(1)
	defer wg.Add(-1)
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
		if quit {
			//该处退出
			return
		}
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
	var tmp []hashData
	//del
	if lastKey != nil {
		if _, ok := mp[string(lastKey)]; ok {
			tmp = mp[string(lastKey)]
			delete(mp, string(lastKey))
		}
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
		delete(mp, key)
	}
	batch.Write()
	//add
	if lastKey != nil {
		if _, ok := mp[string(lastKey)]; ok {
			mp[string(lastKey)] = tmp
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
	var delNodeStrs []string
	var addDelStrs []*addDelStr
	for key := range mp {
		mNode, err := ndb.LoadLeaf([]byte(key))
		if err == nil {
			addDel, delN := mNode.getHashNode(ndb)
			addDelStrs = append(addDelStrs, addDel...)
			delNodeStrs = append(delNodeStrs, delN...)
		}
	}
	//根据keyMap进行归类
	mpAddDel := make(map[string][]addHash)
	for _, s := range addDelStrs {
		key, hash := genDeletePoolKey([]byte(s.str))
		data := addHash{
			hash:  hash,
			isAdd: s.isAdd,
		}
		mpAddDel[string(key)] = append(mpAddDel[string(key)], data)
	}

	//更新pool数据
	batch := db.NewBatch(true)
	for mpk, mpV := range mpAddDel {
		dep := ndb.getPool(mpk)
		if dep != nil {
			for _, aHsh := range mpV {
				if aHsh.isAdd {
					dep.delCache.Add(string(aHsh.hash), true)
				} else {
					dep.delCache.Remove(string(aHsh.hash))
				}
			}
			ndb.updateDelHash(batch, mpk, dep)
		}
	}
	batch.Write()

	//加入要删除的hash节点
	for _, str := range delNodeStrs {
		mp[str] = true
	}
	count := 0
	batch = db.NewBatch(true)
	for key := range mp {
		batch.Delete([]byte(key))
		count++
	}
	batch.Write()
	//fmt.Printf("pruningHashNode ndb.count %d delete %d \n", ndb.count, count1)
	treelog.Info("pruningHashNode ", "delete node count", count)
}

//获取要删除的hash节点
func (node *MarkNode) getHashNode(ndb *markNodeDB) (addDelStrs []*addDelStr, delNodeStrs []string) {
	//目前只裁剪第0,1层节点，暂时不使用递归
	parN := node.fetchParentNode(ndb)
	if parN != nil {
		if bytes.Equal(parN.leftHash, node.hash) {
			node.brotherHash = parN.rightHash
		} else {
			node.brotherHash = parN.leftHash
		}
		broN := node.fetchBrotherNode(ndb)
		if broN == nil || (broN != nil && broN.hashPrune) {
			node.parentPrune = true
		}
		if node.parentPrune {
			if broN == nil {
				//将兄弟节点从已删除map部分移除
				addDelStrs = append(addDelStrs, &addDelStr{false, string(node.brotherHash)})
			}
			//将父节点删除
			delNodeStrs = append(delNodeStrs, string(node.parentHash))
		} else {
			if parN.height == 1 {
				// 需要将本节点加入的已删除map部分,只针对父节点为1,
				// 父节点大于1不处理当前只裁剪叶子节点之上一层
				addDelStrs = append(addDelStrs, &addDelStr{true, string(node.hash)})
			}
		}
	}
	return addDelStrs, delNodeStrs
}

func (ndb *markNodeDB) updateDelHash(batch dbm.Batch, key string, dep *delNodeValuePool) {
	if dep == nil {
		return
	}
	//这里指针暂时不需要赋值
	//if ndb.delPoolCache != nil {
	//	ndb.delPoolCache.Add(key, dep)
	//}
	stp := &types.StoreValuePool{}
	for _, k := range dep.delCache.Keys() {
		stp.Values = append(stp.Values, []byte(k.(string)))
	}
	v, err := proto.Marshal(stp)
	if err != nil {
		panic(fmt.Sprint("types.DeleteNodeMap fail", err))
	}
	batch.Set([]byte(key), v)
}

func (ndb *markNodeDB) getPool(str string) (dep *delNodeValuePool) {
	if ndb.delPoolCache != nil {
		elem, ok := ndb.delPoolCache.Get(str)
		if ok {
			dep = elem.(*delNodeValuePool)
			return dep
		}
	}
	v, err := ndb.db.Get([]byte(str))
	if err != nil || len(v) == 0 {
		//如果不存在说明是新的则
		//创建一个空的集
		ndb.judgeDelNodeCache()
		dep = NewDelNodeValuePool(perDelNodePoolSize)
		if dep != nil {
			ndb.delPoolCache.Add(str, dep)
		}
	} else {
		stp := &types.StoreValuePool{}
		err = proto.Unmarshal(v, stp)
		if err != nil {
			panic(fmt.Sprint("types.StoreValuePool fail", err))
		}
		if ndb.delPoolCache != nil {
			ndb.judgeDelNodeCache()
			dep = NewDelNodeValuePool(perDelNodePoolSize)
			if dep != nil {
				for _, k := range stp.Values {
					dep.delCache.Add(string(k), true)
				}
				ndb.delPoolCache.Add(str, dep)
			}
		}
	}
	return dep
}

func (ndb *markNodeDB) judgeDelNodeCache() {
	if ndb.delPoolCache.Len() >= delNodeCacheSize {
		strs := ndb.delPoolCache.Keys()
		elem, ok := ndb.delPoolCache.Get(strs[0])
		if ok {
			mp := elem.(*delNodeValuePool)
			stp := &types.StoreValuePool{}
			for _, k := range mp.delCache.Keys() {
				stp.Values = append(stp.Values, []byte(k.(string)))
			}
			v, err := proto.Marshal(stp)
			if err != nil {
				panic(fmt.Sprint("types.StoreValuePool fail", err))
			}
			ndb.db.Set([]byte(strs[0].(string)), v)
		}
	}
}

type MarkNode struct {
	height      int32
	hash        []byte
	hashPrune   bool
	leftHash    []byte
	rightHash   []byte
	parentHash  []byte
	parentNode  *MarkNode
	parentPrune bool
	brotherNode *MarkNode
	brotherHash []byte
}

type markNodeDB struct {
	mtx          sync.Mutex
	cache        *lru.Cache // 节点数据缓存
	delPoolCache *lru.Cache // 缓存已经删除的节点hash
	db           dbm.DB
	mLeaf        map[string]bool
}

func newMarkNodeDB(db dbm.DB, cache int, mp map[string]bool) *markNodeDB {
	cach, _ := lru.New(cache)
	ndb := &markNodeDB{
		cache:        cach,
		delPoolCache: delPoolCache,
		db:           db,
		mLeaf:        mp,
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
		mp := ndb.getPool(string(key))
		if mp != nil {
			if _, ok := mp.delCache.Get(string(hsh)); ok {
				return nil, ErrNodeNotExist
			}
		}
		var buf []byte
		buf, err := ndb.db.Get(hash)
		if len(buf) == 0 || err != nil {
			treelog.Debug("----->DeleteNodePool has not this", "hash", common.Bytes2Hex(hash), "err:", err)
			return nil, err
		}
		node, err := MakeNode(buf, nil)
		if err != nil {
			panic(fmt.Sprintf("Error reading IAVLNode. bytes: %X  error: %v", buf, err))
		}
		node.hash = hash
		mNode = &MarkNode{
			height:     node.height,
			hash:       node.hash,
			leftHash:   node.leftHash,
			rightHash:  node.rightHash,
			parentHash: node.parentHash,
		}
		if ndb.cache != nil {
			ndb.cache.Add(string(hash), mNode)
		}
	}
	//检查是否删除
	if _, ok := ndb.mLeaf[string(hash)]; ok {
		mNode.hashPrune = true
	}
	return mNode, nil
}

func PruningTreePrintDB(db dbm.DB, prefix []byte) {
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
					treelog.Debug("pruningTree:", "key:", string(key), "height", pData.Height)
				}
			}
		} else if bytes.Equal(prefix, []byte(hashNodePrefix)) {
			treelog.Debug("pruningTree:", "key:", string(it.Key()))
		} else if bytes.Equal(prefix, []byte(leafNodePrefix)) {
			treelog.Debug("pruningTree:", "key:", string(it.Key()))
		} else if bytes.Equal(prefix, []byte(delMapPoolPrefix)) {
			value := it.Value()
			var pData types.StoreValuePool
			err := proto.Unmarshal(value, &pData)
			if err == nil {
				for _, k := range pData.Values {
					treelog.Debug("delMapPool value ", "hash:", common.Bytes2Hex(k[:2]))
				}
			}
		}
		count++
	}
	fmt.Printf("prefix %s All count:%d \n", string(prefix), count)
	treelog.Info("pruningTree:", "prefix:", string(prefix), "All count", count)
}

func PruningTree(db dbm.DB, curHeight int64) {
	pruningTree(db, curHeight)
}
