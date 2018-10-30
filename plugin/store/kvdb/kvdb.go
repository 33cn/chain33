package kvdb

import (
	"github.com/golang/protobuf/proto"
	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common"
	clog "gitlab.33.cn/chain33/chain33/common/log"
	"gitlab.33.cn/chain33/chain33/queue"
	drivers "gitlab.33.cn/chain33/chain33/system/store"
	"gitlab.33.cn/chain33/chain33/types"
)

var klog = log.New("module", "kvdb")

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	klog.SetHandler(log.DiscardHandler())
}

func init() {
	drivers.Reg("kvdb", New)
}

type KVStore struct {
	*drivers.BaseStore
	cache map[string]map[string]*types.KeyValue
}

func New(cfg *types.Store, sub []byte) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	kvs := &KVStore{bs, make(map[string]map[string]*types.KeyValue)}
	bs.SetChild(kvs)
	return kvs
}

func (kvs *KVStore) Close() {
	kvs.BaseStore.Close()
	klog.Info("store kvdb closed")
}

func (kvs *KVStore) Set(datas *types.StoreSet, sync bool) ([]byte, error) {
	hash := calcHash(datas)
	kvmap := make(map[string]*types.KeyValue)
	for _, kv := range datas.KV {
		kvmap[string(kv.Key)] = kv
	}
	kvs.save(kvmap)
	return hash, nil
}

func (kvs *KVStore) Get(datas *types.StoreGet) [][]byte {
	values := make([][]byte, len(datas.Keys))
	if kvmap, ok := kvs.cache[string(datas.StateHash)]; ok {
		for i := 0; i < len(datas.Keys); i++ {
			kv := kvmap[string(datas.Keys[i])]
			if kv != nil {
				values[i] = kv.Value
			}
		}
	} else {
		db := kvs.GetDB()
		for i := 0; i < len(datas.Keys); i++ {
			value, _ := db.Get(datas.Keys[i])
			if value != nil {
				values[i] = value
			}
		}
	}
	return values
}

func (kvs *KVStore) MemSet(datas *types.StoreSet, sync bool) ([]byte, error) {
	if len(datas.KV) == 0 {
		klog.Info("store kv memset,use preStateHash as stateHash for kvset is null")
		kvmap := make(map[string]*types.KeyValue)
		kvs.cache[string(datas.StateHash)] = kvmap
		return datas.StateHash, nil
	}

	hash := calcHash(datas)
	kvmap := make(map[string]*types.KeyValue)
	for _, kv := range datas.KV {
		kvmap[string(kv.Key)] = kv
	}
	kvs.cache[string(hash)] = kvmap
	if len(kvs.cache) > 100 {
		klog.Error("too many items in cache")
	}
	return hash, nil
}

func (kvs *KVStore) Commit(req *types.ReqHash) ([]byte, error) {
	kvmap, ok := kvs.cache[string(req.Hash)]
	if !ok {
		klog.Error("store kvdb commit", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	if len(kvmap) == 0 {
		klog.Info("store kvdb commit did nothing for kvset is nil")
		delete(kvs.cache, string(req.Hash))
		return req.Hash, nil
	}
	kvs.save(kvmap)
	delete(kvs.cache, string(req.Hash))
	return req.Hash, nil
}

func (kvs *KVStore) Rollback(req *types.ReqHash) ([]byte, error) {
	_, ok := kvs.cache[string(req.Hash)]
	if !ok {
		klog.Error("store kvdb rollback", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	delete(kvs.cache, string(req.Hash))
	return req.Hash, nil
}

func (kvs *KVStore) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	panic("empty")
	//TODO:
	//kvs.IterateRangeByStateHash(mavls.GetDB(), statehash, start, end, ascending, fn)
}

func (kvs *KVStore) ProcEvent(msg queue.Message) {
	msg.ReplyErr("KVStore", types.ErrActionNotSupport)
}

func (kvs *KVStore) Del(req *types.StoreDel) ([]byte, error) {
	//not support
	return nil, nil
}

func (kvs *KVStore) save(kvmap map[string]*types.KeyValue) {
	storeBatch := kvs.GetDB().NewBatch(true)
	for _, kv := range kvmap {
		if kv.Value == nil {
			storeBatch.Delete(kv.Key)
		} else {
			storeBatch.Set(kv.Key, kv.Value)
		}
	}
	storeBatch.Write()
}

func calcHash(datas proto.Message) []byte {
	b := types.Encode(datas)
	return common.Sha256(b)
}
