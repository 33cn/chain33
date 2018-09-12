package kvmvccdb

import (
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/store"
	"gitlab.33.cn/chain33/chain33/types"
)

var klog = log.New("module", "kvmvccdb")

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	klog.SetHandler(log.DiscardHandler())
}

func init() {
	drivers.Reg("kvmvcc", New)
}

type KVMVCCStore struct {
	*drivers.BaseStore
	mvcc     *dbm.SimpleMVCC
	cache    map[string]map[string]*types.KeyValue
	kvsetmap map[string][]*types.KeyValue
}

func New(cfg *types.Store) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	kvs := &KVMVCCStore{bs, dbm.NewSimpleMVCC(dbm.NewKVDB(bs.GetDB())), make(map[string]map[string]*types.KeyValue), make(map[string][]*types.KeyValue)}
	bs.SetChild(kvs)
	return kvs
}

func (mvccs *KVMVCCStore) Close() {
	mvccs.BaseStore.Close()
	klog.Info("store kvdb closed")
}

func (mvccs *KVMVCCStore) Set(datas *types.StoreSet, sync bool) []byte {
	hash := calcHash(datas)

	kvlist, err := mvccs.mvcc.AddMVCC(datas.KV, hash, datas.StateHash, datas.Height)
	if err != nil {
		panic(err)
	}

	klog.Debug("KVMVCCStore Set AddMVCC", "hash", common.ToHex(datas.StateHash), "height", datas.Height)
	//zzh
	for i := 0; i < len(kvlist); i++ {
		klog.Debug("KVMVCCStore Set AddMVCC", "index", i, "key", string(kvlist[i].Key),"value", string(kvlist[i].Value))
	}

	//zzh
	//klog.Debug("KVMVCCStore Set saveKVSets", "hash", common.ToHex(datas.StateHash), "height", datas.Height)
	klog.Info("KVMVCCStore Set saveKVSets", "hash", common.ToHex(datas.StateHash), "height", datas.Height)
	mvccs.saveKVSets(kvlist)
	return hash
}

func (mvccs *KVMVCCStore) Get(datas *types.StoreGet) [][]byte {
	values := make([][]byte, len(datas.Keys))
	if kvmap, ok := mvccs.cache[string(datas.StateHash)]; ok {
		for i := 0; i < len(datas.Keys); i++ {
			kv := kvmap[string(datas.Keys[i])]
			if kv != nil {
				values[i] = kv.Value
				//zzh
				klog.Info("KVMVCCStore Get match cache", "hash", common.HashHex(datas.StateHash), "key", string(datas.Keys[i]), "value", string(values[i]))
				//klog.Debug("KVMVCCStore Get match cache", "hash", common.HashHex(datas.StateHash), "key", string(datas.Keys[i]), "value", string(values[i]))
			}
		}
	} else {
		version, err := mvccs.mvcc.GetVersion(datas.StateHash)
		if err != nil {
			klog.Error("Get version by hash failed.", "hash", common.ToHex(datas.StateHash))
			return values
		}

		//zzh
		klog.Info("KVMVCCStore Get mvcc GetVersion", "hash", common.HashHex(datas.StateHash), "version", version)
		//klog.Debug("KVMVCCStore Get mvcc GetVersion", "hash", common.HashHex(datas.StateHash), "version", version)

		for i := 0; i < len(datas.Keys); i++ {
			value, err := mvccs.mvcc.GetV(datas.Keys[i], version)
			if err != nil {
				klog.Error("GetV by Keys failed.", "Key", string(datas.Keys[i]), "version", version)
			} else if value != nil {
				values[i] = value
				//zzh
				//klog.Debug("KVMVCCStore Get mvcc GetV", "index", i, "key", string(datas.Keys[i]), "value", string(value))
				klog.Info("KVMVCCStore Get mvcc GetV", "index", i, "key", string(datas.Keys[i]), "value", string(value))
			}
		}
	}
	return values
}

func (mvccs *KVMVCCStore) MemSet(datas *types.StoreSet, sync bool) []byte {
	kvset, err := mvccs.checkVersion(datas.Height)
	if err != nil {
		panic(err)
	}

	hash := calcHash(datas)
	//zzh
	//klog.Debug("KVMVCCStore MemSet AddMVCC", "prestatehash", common.ToHex(datas.StateHash), "hash", common.ToHex(hash), "height", datas.Height)
	klog.Info("KVMVCCStore MemSet AddMVCC", "prestatehash", common.ToHex(datas.StateHash), "hash", common.ToHex(hash), "height", datas.Height)
	kvlist, err := mvccs.mvcc.AddMVCC(datas.KV, hash, datas.StateHash, datas.Height)
	if err != nil {
		panic(err)
	}
	/*
		for i := 0; i < len(kvlist); i++ {
			klog.Info("KVMVCCStore MemSet AddMVCC", "index", i, "key", string(kvlist[i].Key),"value", string(kvlist[i].Value))
		}*/
	if len(kvlist) > 0 {
		kvset = append(kvset, kvlist...)
	}
	mvccs.kvsetmap[string(hash)] = kvset

	kvmap := make(map[string]*types.KeyValue)
	for _, kv := range datas.KV {
		kvmap[string(kv.Key)] = kv
	}
	mvccs.cache[string(hash)] = kvmap
	if len(mvccs.cache) > 100 {
		klog.Error("too many items in cache")
	}
	return hash
}

func (mvccs *KVMVCCStore) Commit(req *types.ReqHash) ([]byte, error) {
	_, ok := mvccs.cache[string(req.Hash)]
	if !ok {
		klog.Error("store kvmvcc commit", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	//zzh
	//klog.Debug("KVMVCCStore Commit saveKVSets", "hash", common.ToHex(req.Hash))
	klog.Info("KVMVCCStore Commit saveKVSets", "hash", common.ToHex(req.Hash))
	mvccs.saveKVSets(mvccs.kvsetmap[string(req.Hash)])
	delete(mvccs.cache, string(req.Hash))
	delete(mvccs.kvsetmap, string(req.Hash))
	return req.Hash, nil
}

func (mvccs *KVMVCCStore) Rollback(req *types.ReqHash) []byte {
	_, ok := mvccs.cache[string(req.Hash)]
	if !ok {
		klog.Error("store kvmvcc rollback", "err", types.ErrHashNotFound)
		return nil
	}
	//zzh
	klog.Info("KVMVCCStore Rollback", "hash", common.ToHex(req.Hash))

	delete(mvccs.cache, string(req.Hash))
	delete(mvccs.kvsetmap, string(req.Hash))
	return req.Hash
}

func (mvccs *KVMVCCStore) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	panic("empty")
	//TODO:
	//kvs.IterateRangeByStateHash(mavls.GetDB(), statehash, start, end, ascending, fn)
}

func (mvccs *KVMVCCStore) ProcEvent(msg queue.Message) {
	msg.ReplyErr("KVStore", types.ErrActionNotSupport)
}

func (mvccs *KVMVCCStore) Del(req *types.StoreDel) []byte {
	kvset, err := mvccs.mvcc.DelMVCC(req.StateHash, req.Height)
	if err != nil {
		klog.Error("store kvmvcc del", "err", err)
		return nil
	}

	klog.Info("KVMVCCStore Del", "hash", common.ToHex(req.StateHash), "height", req.Height)
	/*
		for i := 0; i < len(kvset); i++ {
			klog.Debug("KVMVCCStore Del AddMVCC", "index", i, "key", string(kvset[i].Key),"value", string(kvset[i].Value))
		}*/
	mvccs.saveKVSets(kvset)
	return req.StateHash
}

func (mvccs *KVMVCCStore) saveKVSets(kvset []*types.KeyValue) {
	if len(kvset) == 0 {
		return
	}

	storeBatch := mvccs.GetDB().NewBatch(true)

	for i := 0; i < len(kvset); i++ {
		if kvset[i].Value == nil {
			storeBatch.Delete(kvset[i].Key)
		} else {
			storeBatch.Set(kvset[i].Key, kvset[i].Value)
		}
	}
	storeBatch.Write()
}

func (mvccs *KVMVCCStore) checkVersion(height int64) ([]*types.KeyValue, error) {
	//检查新加入区块的height和现有的version的关系，来判断是否要回滚数据
	maxVersion,err := mvccs.mvcc.GetMaxVersion()
	if err != nil {
		klog.Error("store kvmvcc checkVersion GetMaxVersion failed", "err", err)
		panic(err)
	}

	var kvset []*types.KeyValue
	if maxVersion < height - 1 {
		klog.Error("store kvmvcc checkVersion found statehash lost", "maxVersion", maxVersion, "height", height)
		return nil, types.ErrStateHashLost
	} else if maxVersion == height -1 {
		return nil, nil
	} else {
		for i := maxVersion; i >= height; i-- {
			hash, err := mvccs.mvcc.GetVersionHash(i)
			if err != nil {
				klog.Warn("store kvmvcc checkVersion GetVersionHash failed", "height", i)
				continue
			}
			kvlist, err := mvccs.mvcc.DelMVCC(hash, i)
			if err != nil {
				klog.Warn("store kvmvcc checkVersion DelMVCC failed", "height", i, "err", err)
				continue
			}
			kvset = append(kvset, kvlist...)
		}
	}

	return kvset, nil
}

func calcHash(datas proto.Message) []byte {
	b := types.Encode(datas)
	return common.Sha256(b)
}
