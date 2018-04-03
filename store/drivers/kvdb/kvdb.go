package kvdb

import (
	"code.aliyun.com/chain33/chain33/common"
	clog "code.aliyun.com/chain33/chain33/common/log"
	"code.aliyun.com/chain33/chain33/queue"
	"code.aliyun.com/chain33/chain33/store/drivers"
	"code.aliyun.com/chain33/chain33/types"
	log "github.com/inconshreveable/log15"
)

var klog = log.New("module", "kvdb")

func SetLogLevel(level string) {
	clog.SetLogLevel(level)
}

func DisableLog() {
	klog.SetHandler(log.DiscardHandler())
}

type KVStore struct {
	*drivers.BaseStore
	cache map[string]map[string]*types.KeyValue
}

func New(cfg *types.Store) *KVStore {
	bs := drivers.NewBaseStore(cfg)
	kvs := &KVStore{bs, make(map[string]map[string]*types.KeyValue)}
	bs.SetChild(kvs)
	return kvs
}

func (kvs *KVStore) Close() {
	kvs.BaseStore.Close()
	klog.Info("store kvdb closed")
}

func (kvs *KVStore) Set(datas *types.StoreSet) []byte {
	hash := calcHash(datas)
	kvmap := make(map[string]*types.KeyValue)
	for _, kv := range datas.KV {
		kvmap[string(kv.Key)] = kv
	}
	kvs.save(kvmap)
	return hash
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
			value := db.Get(datas.Keys[i])
			if value != nil {
				values[i] = value
			}
		}
	}
	return values
}

func (kvs *KVStore) MemSet(datas *types.StoreSet) []byte {
	hash := calcHash(datas)
	kvmap := make(map[string]*types.KeyValue)
	for _, kv := range datas.KV {
		kvmap[string(kv.Key)] = kv
	}
	kvs.cache[string(hash)] = kvmap
	if len(kvs.cache) > 100 {
		klog.Error("too many items in cache")
	}
	return hash
}

func (kvs *KVStore) Commit(req *types.ReqHash) []byte {
	kvmap, ok := kvs.cache[string(req.Hash)]
	if !ok {
		klog.Error("store kvdb commit", "err", types.ErrHashNotFound)
		return nil
	}
	kvs.save(kvmap)
	delete(kvs.cache, string(req.Hash))
	return req.Hash
}

func (kvs *KVStore) Rollback(req *types.ReqHash) []byte {
	_, ok := kvs.cache[string(req.Hash)]
	if !ok {
		klog.Error("store kvdb rollback", "err", types.ErrHashNotFound)
		return nil
	}
	delete(kvs.cache, string(req.Hash))
	return req.Hash
}

func (kvs *KVStore) ProcEvent(msg queue.Message) {
	msg.ReplyErr("KVStore", types.ErrActionNotSupport)
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

func calcHash(datas *types.StoreSet) []byte {
	b := types.Encode(datas)
	return common.Sha256(b)
}
