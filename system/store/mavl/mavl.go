// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mavl

import (
	"github.com/33cn/chain33/common"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	drivers "github.com/33cn/chain33/system/store"
	mavl "github.com/33cn/chain33/system/store/mavl/db"
	"github.com/33cn/chain33/types"
	lru "github.com/hashicorp/golang-lru"
)

var mlog = log.New("module", "mavl")

// SetLogLevel set log level
func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

// DisableLog disable log
func DisableLog() {
	mlog.SetHandler(log.DiscardHandler())
}

// Store mavl store struct
type Store struct {
	*drivers.BaseStore
	trees            map[string]*mavl.Tree
	cache            *lru.Cache
	enableMavlPrefix bool
	enableMVCC       bool
	enableMavlPrune  bool
	pruneHeight      int32
}

func init() {
	drivers.Reg("mavl", New)
}

type subConfig struct {
	EnableMavlPrefix bool  `json:"enableMavlPrefix"`
	EnableMVCC       bool  `json:"enableMVCC"`
	EnableMavlPrune  bool  `json:"enableMavlPrune"`
	PruneHeight      int32 `json:"pruneHeight"`
}

// New new mavl store module
func New(cfg *types.Store, sub []byte) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	var subcfg subConfig
	if sub != nil {
		types.MustDecode(sub, &subcfg)
	}
	mavls := &Store{bs, make(map[string]*mavl.Tree), nil, subcfg.EnableMavlPrefix, subcfg.EnableMVCC, subcfg.EnableMavlPrune, subcfg.PruneHeight}
	mavls.cache, _ = lru.New(10)
	//使能前缀mavl以及MVCC

	mavls.enableMavlPrefix = subcfg.EnableMavlPrefix
	mavls.enableMVCC = subcfg.EnableMVCC
	mavls.enableMavlPrune = subcfg.EnableMavlPrune
	mavls.pruneHeight = subcfg.PruneHeight
	mavl.EnableMavlPrefix(mavls.enableMavlPrefix)
	mavl.EnableMVCC(mavls.enableMVCC)
	mavl.EnablePrune(mavls.enableMavlPrune)
	mavl.SetPruneHeight(int(mavls.pruneHeight))
	bs.SetChild(mavls)
	return mavls
}

// Close close mavl store
func (mavls *Store) Close() {
	mavl.ClosePrune()
	mavls.BaseStore.Close()
	mlog.Info("store mavl closed")
}

// Set set k v to mavl store db; sync is true represent write sync
func (mavls *Store) Set(datas *types.StoreSet, sync bool) ([]byte, error) {
	return mavl.SetKVPair(mavls.GetDB(), datas, sync)
}

// Get get values by keys
func (mavls *Store) Get(datas *types.StoreGet) [][]byte {
	var tree *mavl.Tree
	var err error
	values := make([][]byte, len(datas.Keys))
	search := string(datas.StateHash)
	if data, ok := mavls.cache.Get(search); ok {
		tree = data.(*mavl.Tree)
	} else if data, ok := mavls.trees[search]; ok {
		tree = data
	} else {
		tree = mavl.NewTree(mavls.GetDB(), true)
		//get接口也应该传入高度
		//tree.SetBlockHeight(datas.Height)
		err = tree.Load(datas.StateHash)
		if err == nil {
			mavls.cache.Add(search, tree)
		}
		mlog.Debug("store mavl get tree", "err", err, "StateHash", common.ToHex(datas.StateHash))
	}
	if err == nil {
		for i := 0; i < len(datas.Keys); i++ {
			_, value, exit := tree.Get(datas.Keys[i])
			if exit {
				values[i] = value
			}
		}
	}
	return values
}

// MemSet set keys values to memcory mavl, return root hash and error
func (mavls *Store) MemSet(datas *types.StoreSet, sync bool) ([]byte, error) {
	if len(datas.KV) == 0 {
		mlog.Info("store mavl memset,use preStateHash as stateHash for kvset is null")
		mavls.trees[string(datas.StateHash)] = nil
		return datas.StateHash, nil
	}
	tree := mavl.NewTree(mavls.GetDB(), sync)
	tree.SetBlockHeight(datas.Height)
	err := tree.Load(datas.StateHash)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(datas.KV); i++ {
		tree.Set(datas.KV[i].Key, datas.KV[i].Value)
	}
	hash := tree.Hash()
	mavls.trees[string(hash)] = tree
	if len(mavls.trees) > 1000 {
		mlog.Error("too many trees in cache")
	}
	return hash, nil
}

// Commit convert memcory mavl to storage db
func (mavls *Store) Commit(req *types.ReqHash) ([]byte, error) {
	tree, ok := mavls.trees[string(req.Hash)]
	if !ok {
		mlog.Error("store mavl commit", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}

	if tree == nil {
		mlog.Info("store mavl commit,do nothing for kvset is null")
		delete(mavls.trees, string(req.Hash))
		return req.Hash, nil
	}

	hash := tree.Save()
	if hash == nil {
		mlog.Error("store mavl commit", "err", types.ErrHashNotFound)
		return nil, types.ErrDataBaseDamage
	}
	delete(mavls.trees, string(req.Hash))
	return req.Hash, nil
}

// Rollback 回退将缓存的mavl树删除掉
func (mavls *Store) Rollback(req *types.ReqHash) ([]byte, error) {
	_, ok := mavls.trees[string(req.Hash)]
	if !ok {
		mlog.Error("store mavl rollback", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	delete(mavls.trees, string(req.Hash))
	return req.Hash, nil
}

// IterateRangeByStateHash 迭代实现功能； statehash：当前状态hash, start：开始查找的key, end: 结束的key, ascending：升序，降序, fn 迭代回调函数
func (mavls *Store) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	mavl.IterateRangeByStateHash(mavls.GetDB(), statehash, start, end, ascending, fn)
}

// ProcEvent not support message
func (mavls *Store) ProcEvent(msg queue.Message) {
	msg.ReplyErr("Store", types.ErrActionNotSupport)
}

// Del ...
func (mavls *Store) Del(req *types.StoreDel) ([]byte, error) {
	//not support
	return nil, nil
}
