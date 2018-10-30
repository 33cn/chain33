package mpt

import (
	"github.com/hashicorp/golang-lru"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	mpt "gitlab.33.cn/chain33/chain33/plugin/store/mpt/db"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/store"
	"gitlab.33.cn/chain33/chain33/types"
)

var mlog = log.New("module", "mpt")

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	mlog.SetHandler(log.DiscardHandler())
}

type Store struct {
	*drivers.BaseStore
	trees map[string]*mpt.TrieEx
	cache *lru.Cache
}

func init() {
	drivers.Reg("mpt", New)
}

func New(cfg *types.Store, sub []byte) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	mpts := &Store{bs, make(map[string]*mpt.TrieEx), nil}
	mpts.cache, _ = lru.New(10)
	bs.SetChild(mpts)
	return mpts
}

func (mpts *Store) Close() {
	mpts.BaseStore.Close()
	mlog.Info("store mavl closed")
}

func (mpts *Store) Set(datas *types.StoreSet, sync bool) ([]byte, error) {
	hash, err := mpt.SetKVPair(mpts.GetDB(), datas, sync)
	if err != nil {
		mlog.Error("mpt store error", "err", err)
		return nil, err
	}
	return hash, nil
}

func (mpts *Store) Get(datas *types.StoreGet) [][]byte {
	var tree *mpt.TrieEx
	var err error
	values := make([][]byte, len(datas.Keys))
	search := string(datas.StateHash)
	if data, ok := mpts.cache.Get(search); ok {
		tree = data.(*mpt.TrieEx)
	} else if data, ok := mpts.trees[search]; ok {
		tree = data
	} else {
		tree, err = mpt.NewEx(common.BytesToHash(datas.StateHash), mpt.NewDatabase(mpts.GetDB()))
		if nil != err {
			mlog.Error("Store get can not find a trie")
		}
		if nil == err {
			mpts.cache.Add(search, tree)
		}
		mlog.Debug("store mpt get tree", "err", err, "StateHash", common.ToHex(datas.StateHash))
	}
	if err == nil {
		for i := 0; i < len(datas.Keys); i++ {
			value, err := tree.TryGet(datas.Keys[i])
			if nil == err {
				values[i] = value
			}
		}
	}
	return values
}

func (mpts *Store) MemSet(datas *types.StoreSet, sync bool) ([]byte, error) {
	var err error
	var tree *mpt.TrieEx
	tree, err = mpt.NewEx(common.BytesToHash(datas.StateHash), mpt.NewDatabase(mpts.GetDB()))
	if err != nil {
		mlog.Info("MemSet create a new trie", "err", err)
		return nil, err
	}
	for i := 0; i < len(datas.KV); i++ {
		tree.Update(datas.KV[i].Key, datas.KV[i].Value)
	}
	root, err := tree.Commit(nil)
	if err != nil {
		mlog.Error("MemSet Commit to memory trie fail")
		return nil, err
	}
	hash := root[:]
	mpts.trees[string(hash)] = tree
	if len(mpts.trees) > 1000 {
		mlog.Error("too many trees in cache")
	}
	return hash, nil
}

func (mpts *Store) Commit(req *types.ReqHash) ([]byte, error) {
	tree, ok := mpts.trees[string(req.Hash)]
	if !ok {
		mlog.Error("store mpt commit", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	err := tree.Commit2Db(common.BytesToHash(req.Hash), true)
	if nil != err {
		mlog.Error("store mpt commit", "err", types.ErrHashNotFound)
		return nil, types.ErrDataBaseDamage
	}
	delete(mpts.trees, string(req.Hash))
	return req.Hash, nil
}

func (mpts *Store) Rollback(req *types.ReqHash) ([]byte, error) {
	_, ok := mpts.trees[string(req.Hash)]
	if !ok {
		mlog.Error("store mavl rollback", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	delete(mpts.trees, string(req.Hash))
	return req.Hash, nil
}

func (mpts *Store) Del(req *types.StoreDel) ([]byte, error) {
	//not support
	return nil, nil
}

func (mpts *Store) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	mpt.IterateRangeByStateHash(mpts.GetDB(), statehash, start, end, ascending, fn)
}

func (mpts *Store) ProcEvent(msg queue.Message) {
	msg.ReplyErr("Store", types.ErrActionNotSupport)
}
