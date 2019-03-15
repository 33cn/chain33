// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"bytes"
	"fmt"
	"sync"

	"runtime"
	"sync/atomic"
	"time"

	"strconv"

	"github.com/33cn/chain33/common"
	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/golang/protobuf/proto"
)

const (
	leafKeyCountPrefix     = "..mk.."
	oldLeafKeyCountPrefix  = "..mok.."
	secLvlPruningHeightKey = "_..mslphk.._"
	blockHeightStrLen      = 10
	hashLenStr             = 3
	pruningStateStart      = 1
	pruningStateEnd        = 0
	//二级裁剪高度，达到此高度未裁剪则放入该处
	secondLevelPruningHeight = 500000
	//三级裁剪高度，达到此高度还没有裁剪，则不进行裁剪
	threeLevelPruningHeight = 1500000
	onceScanCount           = 10000 // 单次扫描数目
	onceCount               = 1000  // 容器长度
	batchDataSize           = 1024 * 1024 * 1
)

var (
	// 是否开启mavl裁剪
	enablePrune bool
	// 每个10000裁剪一次
	pruneHeight = 10000
	// 裁剪状态
	pruningState   int32
	wg             sync.WaitGroup
	quit           bool
	secLvlPruningH int64
)

type hashData struct {
	height int64
	hash   []byte
}

// EnablePrune 使能裁剪
func EnablePrune(enable bool) {
	enablePrune = enable
	//开启裁剪需要同时开启前缀
	enableMavlPrefix = enable
}

// SetPruneHeight 设置每次裁剪高度
func SetPruneHeight(height int) {
	pruneHeight = height
}

// ClosePrune 关闭裁剪
func ClosePrune() {
	quit = true
	wg.Wait()
	//防止BaseStore没有关闭再次进入
	setPruning(pruningStateStart)
}

func genLeafCountKey(key, hash []byte, height int64, hashLen int) (hashkey []byte) {
	hashkey = []byte(fmt.Sprintf("%s%s%010d%s%03d", leafKeyCountPrefix, string(key), height, string(hash), hashLen))
	return hashkey
}

func getKeyHeightFromLeafCountKey(hashkey []byte) (key []byte, height int, hash []byte, err error) {
	if len(hashkey) < len(leafKeyCountPrefix)+blockHeightStrLen+sha256Len+hashLenStr {
		return nil, 0, nil, types.ErrSize
	}
	if !bytes.Contains(hashkey, []byte(leafKeyCountPrefix)) {
		return nil, 0, nil, types.ErrSize
	}
	sLen := hashkey[len(hashkey)-hashLenStr:]
	iLen, err := strconv.Atoi(string(sLen))
	if err != nil {
		return nil, 0, nil, types.ErrSize
	}
	k := bytes.TrimPrefix(hashkey, []byte(leafKeyCountPrefix))
	key = k[:len(k)-iLen-blockHeightStrLen-hashLenStr]
	//keyHeighthash
	heightHash := k[len(key) : len(k)-hashLenStr]
	sHeight := heightHash[:blockHeightStrLen]
	height, err = strconv.Atoi(string(sHeight))
	if err != nil {
		return nil, 0, nil, types.ErrSize
	}
	hash = heightHash[blockHeightStrLen:]
	return key, height, hash, nil
}

func genOldLeafCountKey(key, hash []byte, height int64, hashLen int) (hashkey []byte) {
	hashkey = []byte(fmt.Sprintf("%s%s%010d%s%03d", oldLeafKeyCountPrefix, string(key), height, string(hash), hashLen))
	return hashkey
}

func getKeyHeightFromOldLeafCountKey(hashkey []byte) (key []byte, height int, hash []byte, err error) {
	if len(hashkey) < len(oldLeafKeyCountPrefix)+blockHeightStrLen+sha256Len+hashLenStr {
		return nil, 0, nil, types.ErrSize
	}
	if !bytes.Contains(hashkey, []byte(oldLeafKeyCountPrefix)) {
		return nil, 0, nil, types.ErrSize
	}
	sLen := hashkey[len(hashkey)-hashLenStr:]
	iLen, err := strconv.Atoi(string(sLen))
	if err != nil {
		return nil, 0, nil, types.ErrSize
	}
	k := bytes.TrimPrefix(hashkey, []byte(oldLeafKeyCountPrefix))
	key = k[:len(k)-iLen-blockHeightStrLen-hashLenStr]
	//keyHeighthash
	heightHash := k[len(key) : len(k)-hashLenStr]
	sHeight := heightHash[:blockHeightStrLen]
	height, err = strconv.Atoi(string(sHeight))
	if err != nil {
		return nil, 0, nil, types.ErrSize
	}
	hash = heightHash[blockHeightStrLen:]
	return key, height, hash, nil
}

func genOldLeafCountKeyFromKey(hashk []byte) (oldhashk []byte) {
	if len(hashk) < len(leafKeyCountPrefix) {
		return hashk
	}
	oldhashk = []byte(fmt.Sprintf("%s%s", oldLeafKeyCountPrefix, string(hashk[len(leafKeyCountPrefix):])))
	return oldhashk
}

func isPruning() bool {
	return atomic.LoadInt32(&pruningState) == 1
}

func setPruning(state int32) {
	atomic.StoreInt32(&pruningState, state)
}

func getSecLvlPruningHeight(db dbm.DB) int64 {
	value, err := db.Get([]byte(secLvlPruningHeightKey))
	if len(value) == 0 || err != nil {
		return 0
	}
	h := &types.Int64{}
	err = proto.Unmarshal(value, h)
	if err != nil {
		return 0
	}
	return h.Data
}

func setSecLvlPruningHeight(db dbm.DB, height int64) error {
	h := &types.Int64{}
	h.Data = height
	value, err := proto.Marshal(h)
	if err != nil {
		return err
	}
	return db.Set([]byte(secLvlPruningHeightKey), value)
}

func pruning(db dbm.DB, curHeight int64) {
	defer wg.Done()
	pruningTree(db, curHeight)
}

func pruningTree(db dbm.DB, curHeight int64) {
	setPruning(pruningStateStart)
	// 一级遍历
	pruningFirstLevel(db, curHeight)
	// 二级遍历
	pruningSecondLevel(db, curHeight)
	setPruning(pruningStateEnd)
}

func pruningFirstLevel(db dbm.DB, curHeight int64) {
	treelog.Info("pruningTree pruningFirstLevel", "start curHeight:", curHeight)
	start := time.Now()
	pruningFirstLevelNode(db, curHeight)
	end := time.Now()
	treelog.Info("pruningTree pruningFirstLevel", "curHeight:", curHeight, "pruning leafNode cost time:", end.Sub(start))
}

func pruningFirstLevelNode(db dbm.DB, curHeight int64) {
	prefix := []byte(leafKeyCountPrefix)
	it := db.Iterator(prefix, nil, true)
	defer it.Close()

	var mp map[string][]hashData
	var kvs []*types.KeyValue
	count := 0
	batch := db.NewBatch(true)
	for it.Rewind(); it.Valid(); it.Next() {
		if quit {
			//该处退出
			return
		}
		if mp == nil {
			mp = make(map[string][]hashData, onceCount)
		}
		//copy key
		hashK := make([]byte, len(it.Key()))
		copy(hashK, it.Key())

		key, height, hash, err := getKeyHeightFromLeafCountKey(hashK)
		if err != nil {
			continue
		}
		if curHeight < int64(height)+secondLevelPruningHeight {
			if curHeight >= int64(height)+int64(pruneHeight) {
				data := hashData{
					height: int64(height),
					hash:   hash,
				}
				mp[string(key)] = append(mp[string(key)], data)
				count++
			}
		} else {
			value := make([]byte, len(it.Value()))
			copy(value, it.Value())
			kvs = append(kvs, &types.KeyValue{Key: hashK, Value: value})
		}
		if len(mp) >= onceCount-1 || count > onceScanCount {
			deleteNode(db, mp, curHeight, batch)
			mp = nil
			count = 0
		}
		if len(kvs) >= onceCount {
			addLeafCountKeyToSecondLevel(db, kvs, batch)
			kvs = kvs[:0]
		}
	}
	if len(mp) > 0 {
		deleteNode(db, mp, curHeight, batch)
		mp = nil
		_ = mp
	}
	if len(kvs) > 0 {
		addLeafCountKeyToSecondLevel(db, kvs, batch)
	}
}

func addLeafCountKeyToSecondLevel(db dbm.DB, kvs []*types.KeyValue, batch dbm.Batch) {
	var err error
	batch.Reset()
	for _, kv := range kvs {
		batch.Delete(kv.Key)
		batch.Set(genOldLeafCountKeyFromKey(kv.Key), kv.Value)
		if batch.ValueSize() > batchDataSize {
			if err = batch.Write(); err != nil {
				return
			}
			batch.Reset()
		}
	}
	if err = batch.Write(); err != nil {
		return
	}
}

func deleteNode(db dbm.DB, mp map[string][]hashData, curHeight int64, batch dbm.Batch) {
	if len(mp) == 0 {
		return
	}
	var err error
	batch.Reset()
	for key, vals := range mp {
		if len(vals) > 1 && vals[1].height != vals[0].height { //防止相同高度时候出现的误删除
			for _, val := range vals[1:] { //从第二个开始判断
				if curHeight >= val.height+int64(pruneHeight) {
					leafCountKey := genLeafCountKey([]byte(key), val.hash, val.height, len(val.hash))
					value, err := db.Get(leafCountKey)
					if err == nil {
						var pData types.PruneData
						err := proto.Unmarshal(value, &pData)
						if err == nil {
							for _, hash := range pData.Hashs {
								batch.Delete(hash)
							}
						}
					}
					batch.Delete(leafCountKey) // 叶子计数节点
					batch.Delete(val.hash)     // 叶子节点hash值
					if batch.ValueSize() > batchDataSize {
						if err = batch.Write(); err != nil {
							return
						}
						batch.Reset()
					}
				}
			}
		}
		delete(mp, key)
	}
	if err = batch.Write(); err != nil {
		return
	}
}

func pruningSecondLevel(db dbm.DB, curHeight int64) {
	if secLvlPruningH == 0 {
		secLvlPruningH = getSecLvlPruningHeight(db)
	}
	if curHeight/secondLevelPruningHeight > 1 &&
		curHeight/secondLevelPruningHeight != secLvlPruningH/secondLevelPruningHeight {
		treelog.Info("pruningTree pruningSecondLevel", "start curHeight:", curHeight)
		start := time.Now()
		pruningSecondLevelNode(db, curHeight)
		end := time.Now()
		treelog.Info("pruningTree pruningSecondLevel", "curHeight:", curHeight, "pruning leafNode cost time:", end.Sub(start))
		err := setSecLvlPruningHeight(db, curHeight)
		if err != nil {
			return
		}
		secLvlPruningH = curHeight
	}
}

func pruningSecondLevelNode(db dbm.DB, curHeight int64) {
	prefix := []byte(oldLeafKeyCountPrefix)
	it := db.Iterator(prefix, nil, true)
	defer it.Close()

	var mp map[string][]hashData
	count := 0
	batch := db.NewBatch(true)
	for it.Rewind(); it.Valid(); it.Next() {
		if quit {
			//该处退出
			return
		}
		if mp == nil {
			mp = make(map[string][]hashData, onceCount)
		}
		//copy key
		hashK := make([]byte, len(it.Key()))
		copy(hashK, it.Key())
		key, height, hash, err := getKeyHeightFromOldLeafCountKey(hashK)
		if err == nil {
			data := hashData{
				height: int64(height),
				hash:   hash,
			}
			mp[string(key)] = append(mp[string(key)], data)
			count++
			if len(mp) >= onceCount-1 || count > onceScanCount {
				deleteOldNode(db, mp, curHeight, batch)
				mp = nil
				count = 0
			}
		}
	}
	if len(mp) > 0 {
		deleteOldNode(db, mp, curHeight, batch)
		mp = nil
		_ = mp
	}
}

func deleteOldNode(db dbm.DB, mp map[string][]hashData, curHeight int64, batch dbm.Batch) {
	if len(mp) == 0 {
		return
	}
	var err error
	batch.Reset()
	for key, vals := range mp {
		if len(vals) > 1 {
			if vals[1].height != vals[0].height { //防止相同高度时候出现的误删除
				for _, val := range vals[1:] { //从第二个开始判断
					if curHeight >= val.height+int64(pruneHeight) {
						leafCountKey := genOldLeafCountKey([]byte(key), val.hash, val.height, len(val.hash))
						value, err := db.Get(leafCountKey)
						if err == nil {
							var pData types.PruneData
							err := proto.Unmarshal(value, &pData)
							if err == nil {
								for _, hash := range pData.Hashs {
									batch.Delete(hash)
								}
							}
						}
						batch.Delete(leafCountKey)
						batch.Delete(val.hash) // 叶子节点hash值
					}
				}
			} else {
				// 删除第三层存储索引key
				for _, val := range vals {
					if curHeight >= val.height+threeLevelPruningHeight {
						batch.Delete(genOldLeafCountKey([]byte(key), val.hash, val.height, len(val.hash)))
					}
				}
			}
		} else if len(vals) == 1 && curHeight >= vals[0].height+threeLevelPruningHeight { // 删除第三层存储索引key
			batch.Delete(genOldLeafCountKey([]byte(key), vals[0].hash, vals[0].height, len(vals[0].hash)))
		}
		delete(mp, key)
		if batch.ValueSize() > batchDataSize {
			if err = batch.Write(); err != nil {
				return
			}
			batch.Reset()
		}
	}
	if err = batch.Write(); err != nil {
		return
	}
}

// PruningTreePrintDB pruning tree print db
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
				key, height, _, err := getKeyHeightFromLeafCountKey(hashK)
				if err == nil {
					treelog.Debug("pruningTree:", "key:", string(key), "height", height)
				}
			}
		} else if bytes.Equal(prefix, []byte(hashNodePrefix)) {
			treelog.Debug("pruningTree:", "key:", string(it.Key()))
		} else if bytes.Equal(prefix, []byte(leafNodePrefix)) {
			treelog.Debug("pruningTree:", "key:", string(it.Key()))
		}
		count++
	}
	fmt.Printf("prefix %s All count:%d \n", string(prefix), count)
	treelog.Info("pruningTree:", "prefix:", string(prefix), "All count", count)
}

// PrintSameLeafKey 查询相同的叶子节点
func PrintSameLeafKey(db dbm.DB, key string) {
	printSameLeafKeyDB(db, key, false)
	printSameLeafKeyDB(db, key, true)
}

func printSameLeafKeyDB(db dbm.DB, key string, isold bool) {
	var prifex string
	if isold {
		prifex = oldLeafKeyCountPrefix
	} else {
		prifex = leafKeyCountPrefix
	}
	priKey := []byte(fmt.Sprintf("%s%s", prifex, key))

	it := db.Iterator(priKey, nil, true)
	defer it.Close()

	for it.Rewind(); it.Valid(); it.Next() {
		hashK := make([]byte, len(it.Key()))
		copy(hashK, it.Key())

		key, height, hash, err := getKeyHeightFromLeafCountKey(hashK)
		if err == nil {
			pri := ""
			if len(hash) > 32 {
				pri = string(hash[:16])
			}
			treelog.Info("leaf node", "height", height, "pri", pri, "hash", common.ToHex(hash), "key", string(key))
		}
	}
}

// PrintLeafNodeParent 查询叶子节点的父节点
func PrintLeafNodeParent(db dbm.DB, key, hash []byte, height int64) {
	var isHave bool
	leafCountKey := genLeafCountKey(key, hash, height, len(hash))
	value, err := db.Get(leafCountKey)
	if err == nil {
		var pData types.PruneData
		err := proto.Unmarshal(value, &pData)
		if err == nil {
			for _, hash := range pData.Hashs {
				var pri string
				if len(hash) > 32 {
					pri = string(hash[:16])
				}
				treelog.Info("hash node", "hash pri", pri, "hash", common.ToHex(hash))
			}
		}
		isHave = true
	}

	if !isHave {
		oldLeafCountKey := genOldLeafCountKey(key, hash, height, len(hash))
		value, err = db.Get(oldLeafCountKey)
		if err == nil {
			var pData types.PruneData
			err := proto.Unmarshal(value, &pData)
			if err == nil {
				for _, hash := range pData.Hashs {
					var pri string
					if len(hash) > 32 {
						pri = string(hash[:16])
					}
					treelog.Info("hash node", "hash pri", pri, "hash", common.ToHex(hash))
				}
			}
			isHave = true
		}
	}

	if !isHave {
		treelog.Info("err", "get db", "not exist in db")
	}
}

// PrintNodeDb 查询hash节点以及其子节点
func PrintNodeDb(db dbm.DB, hash []byte) {
	nDb := newNodeDB(db, true)
	node, err := nDb.GetNode(nil, hash)
	if err != nil {
		fmt.Println("err", err)
		return
	}
	pri := ""
	if len(node.hash) > 32 {
		pri = string(node.hash[:16])
	}
	treelog.Info("hash node", "hash pri", pri, "hash", common.ToHex(node.hash), "height", node.height)
	node.printNodeInfo(nDb)
}

func (node *Node) printNodeInfo(db *nodeDB) {
	if node.height == 0 {
		pri := ""
		if len(node.hash) > 32 {
			pri = string(node.hash[:16])
		}
		treelog.Info("leaf node", "hash pri", pri, "hash", common.ToHex(node.hash), "key", string(node.key))
		return
	}
	if node.leftHash != nil {
		left, err := db.GetNode(nil, node.leftHash)
		if err != nil {
			return
		}
		pri := ""
		if len(left.hash) > 32 {
			pri = string(left.hash[:16])
		}
		treelog.Debug("hash node", "hash pri", pri, "hash", common.ToHex(left.hash), "height", left.height)
		left.printNodeInfo(db)
	}

	if node.rightHash != nil {
		right, err := db.GetNode(nil, node.rightHash)
		if err != nil {
			return
		}
		pri := ""
		if len(right.hash) > 32 {
			pri = string(right.hash[:16])
		}
		treelog.Debug("hash node", "hash pri", pri, "hash", common.ToHex(right.hash), "height", right.height)
		right.printNodeInfo(db)
	}
}

// PruningTree 裁剪树
func PruningTree(db dbm.DB, curHeight int64) {
	pruningTree(db, curHeight)
}

// PrintMemStats 打印内存使用情况
func PrintMemStats(height int64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	treelog.Info("printMemStats:", "程序向系统申请", m.HeapSys/(1024*1024), "堆上目前分配Alloc:", m.HeapAlloc/(1024*1024), "堆上没有使用", m.HeapIdle/(1024*1024), "HeapReleased", m.HeapReleased/(1024*1024), "height", height)
}
