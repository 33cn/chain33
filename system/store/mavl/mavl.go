// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package mavl 默克尔平衡树接口
package mavl

import (
	"sync"

	"github.com/33cn/chain33/common"
	clog "github.com/33cn/chain33/common/log"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/queue"
	drivers "github.com/33cn/chain33/system/store"
	mavl "github.com/33cn/chain33/system/store/mavl/db"
	"github.com/33cn/chain33/types"
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
	trees   *sync.Map
	treeCfg *mavl.TreeConfig
}

func init() {
	drivers.Reg("mavl", New)
}

type subConfig struct {
	// 是否使能mavl加前缀
	EnableMavlPrefix bool `json:"enableMavlPrefix"`
	EnableMVCC       bool `json:"enableMVCC"`
	// 是否使能mavl数据裁剪
	EnableMavlPrune bool `json:"enableMavlPrune"`
	// 裁剪高度间隔
	PruneHeight int32 `json:"pruneHeight"`
	// 是否使能内存树
	EnableMemTree bool `json:"enableMemTree"`
	// 是否使能内存树中叶子节点
	EnableMemVal bool `json:"enableMemVal"`
	// 缓存close ticket数目
	TkCloseCacheLen int32 `json:"tkCloseCacheLen"`
}

// New new mavl store module
func New(cfg *types.Store, sub []byte, chain33cfg *types.Chain33Config) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	var subcfg subConfig
	if sub != nil {
		types.MustDecode(sub, &subcfg)
	}
	// 开启裁剪需要同时开启前缀树
	if subcfg.EnableMavlPrune {
		subcfg.EnableMavlPrefix = subcfg.EnableMavlPrune
	}
	treeCfg := &mavl.TreeConfig{
		EnableMavlPrefix: subcfg.EnableMavlPrefix,
		EnableMVCC:       subcfg.EnableMVCC,
		EnableMavlPrune:  subcfg.EnableMavlPrune,
		PruneHeight:      subcfg.PruneHeight,
		EnableMemTree:    subcfg.EnableMemTree,
		EnableMemVal:     subcfg.EnableMemVal,
		TkCloseCacheLen:  subcfg.TkCloseCacheLen,
	}
	mavls := &Store{bs, &sync.Map{}, treeCfg}
	mavl.InitGlobalMem(treeCfg)
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
	return mavl.SetKVPair(mavls.GetDB(), datas, sync, mavls.treeCfg)
}

// Get get values by keys
func (mavls *Store) Get(datas *types.StoreGet) [][]byte {
	var tree *mavl.Tree
	var err error
	values := make([][]byte, len(datas.Keys))
	search := string(datas.StateHash)
	if data, ok := mavls.trees.Load(search); ok && data != nil {
		tree = data.(*mavl.Tree)
	} else {
		tree = mavl.NewTree(mavls.GetDB(), true, mavls.treeCfg)
		//get接口也应该传入高度
		//tree.SetBlockHeight(datas.Height)
		err = tree.Load(datas.StateHash)
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
	beg := types.Now()
	defer func() {
		mlog.Debug("MemSet", "cost", types.Since(beg))
	}()
	if len(datas.KV) == 0 {
		mlog.Info("store mavl memset,use preStateHash as stateHash for kvset is null")
		mavls.trees.Store(string(datas.StateHash), nil)
		return datas.StateHash, nil
	}
	tree := mavl.NewTree(mavls.GetDB(), sync, mavls.treeCfg)
	tree.SetBlockHeight(datas.Height)
	err := tree.Load(datas.StateHash)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(datas.KV); i++ {
		tree.Set(datas.KV[i].Key, datas.KV[i].Value)
	}
	hash := tree.Hash()
	mavls.trees.Store(string(hash), tree)
	return hash, nil
}

// Commit convert memcory mavl to storage db
func (mavls *Store) Commit(req *types.ReqHash) ([]byte, error) {
	beg := types.Now()
	defer func() {
		mlog.Debug("Commit", "cost", types.Since(beg))
	}()
	tree, ok := mavls.trees.Load(string(req.Hash))
	if !ok {
		mlog.Error("store mavl commit", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	if tree == nil {
		mlog.Info("store mavl commit,do nothing for kvset is null")
		mavls.trees.Delete(string(req.Hash))
		return req.Hash, nil
	}
	hash := tree.(*mavl.Tree).Save()
	if hash == nil {
		mlog.Error("store mavl commit", "err", types.ErrHashNotFound)
		return nil, types.ErrDataBaseDamage
	}
	mavls.trees.Delete(string(req.Hash))
	return req.Hash, nil
}

// MemSetUpgrade cacl mavl, but not store tree, return root hash and error
func (mavls *Store) MemSetUpgrade(datas *types.StoreSet, sync bool) ([]byte, error) {
	beg := types.Now()
	defer func() {
		mlog.Debug("MemSet", "cost", types.Since(beg))
	}()
	if len(datas.KV) == 0 {
		mlog.Info("store mavl memset,use preStateHash as stateHash for kvset is null")
		mavls.trees.Store(string(datas.StateHash), nil)
		return datas.StateHash, nil
	}
	tree := mavl.NewTree(mavls.GetDB(), sync, mavls.treeCfg)
	tree.SetBlockHeight(datas.Height)
	err := tree.Load(datas.StateHash)
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(datas.KV); i++ {
		tree.Set(datas.KV[i].Key, datas.KV[i].Value)
	}
	hash := tree.Hash()
	return hash, nil
}

// CommitUpgrade convert memcory mavl to storage db
func (mavls *Store) CommitUpgrade(req *types.ReqHash) ([]byte, error) {
	return req.Hash, nil
}

// Rollback 回退将缓存的mavl树删除掉
func (mavls *Store) Rollback(req *types.ReqHash) ([]byte, error) {
	beg := types.Now()
	defer func() {
		mlog.Debug("Rollback", "cost", types.Since(beg))
	}()
	_, ok := mavls.trees.Load(string(req.Hash))
	if !ok {
		mlog.Error("store mavl rollback", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	mavls.trees.Delete(string(req.Hash))
	return req.Hash, nil
}

// IterateRangeByStateHash 迭代实现功能； statehash：当前状态hash, start：开始查找的key, end: 结束的key, ascending：升序，降序, fn 迭代回调函数
func (mavls *Store) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	mavl.IterateRangeByStateHash(mavls.GetDB(), statehash, start, end, ascending, mavls.treeCfg, fn)
}

// ProcEvent not support message
func (mavls *Store) ProcEvent(msg *queue.Message) {
	if msg == nil {
		return
	}
	msg.ReplyErr("Store", types.ErrActionNotSupport)
}

// Del ...
func (mavls *Store) Del(req *types.StoreDel) ([]byte, error) {
	//not support
	return nil, nil
}
