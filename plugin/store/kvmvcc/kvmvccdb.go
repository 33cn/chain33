package kvmvccdb

import (
	"sync/atomic"

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

const (
	FlagInit        = int64(0)
	FlagFromZero    = int64(1)
	FlagNotFromZero = int64(2)
)

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
	flagMVCC int64
	cache    map[string]map[string]*types.KeyValue
	kvsetmap map[string][]*types.KeyValue
}

func New(cfg *types.Store) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	kvs := &KVMVCCStore{bs, dbm.NewSimpleMVCC(dbm.NewKVDB(bs.GetDB())), FlagInit, make(map[string]map[string]*types.KeyValue), make(map[string][]*types.KeyValue)}
	bs.SetChild(kvs)
	return kvs
}

func (mvccs *KVMVCCStore) Close() {
	mvccs.BaseStore.Close()
	klog.Info("store kvdb closed")
}

func (mvccs *KVMVCCStore) Set(datas *types.StoreSet, sync bool) []byte {
	kvset, err := mvccs.checkMVCCFlag(mvccs.GetDB(), datas.Height)
	if err != nil {
		panic(err)
	}
	klog.Debug("KVMVCCStore Set checkMVCCFlag", "checkMVCCFlag", kvset)
	hash := calcHash(datas)

	kvlist, err := mvccs.mvcc.AddMVCC(datas.KV, hash, datas.StateHash, datas.Height)
	if err != nil {
		panic(err)
	}

	klog.Debug("KVMVCCStore Set AddMVCC", "hash", common.ToHex(datas.StateHash), "height", datas.Height)
	/*
		for i := 0; i < len(kvlist); i++ {
			klog.Debug("KVMVCCStore Set AddMVCC", "index", i, "key", string(kvlist[i].Key),"value", string(kvlist[i].Value))
		}*/

	if len(kvset) > 0 {
		kvlist = append(kvlist, kvset...)
	}

	klog.Debug("KVMVCCStore Set saveKVSets", "hash", common.ToHex(datas.StateHash), "height", datas.Height)
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
				klog.Debug("KVMVCCStore Get match cache", "hash", common.HashHex(datas.StateHash), "key", string(datas.Keys[i]), "value", string(values[i]))
			}
		}
	} else {
		version, err := mvccs.mvcc.GetVersion(datas.StateHash)
		if err != nil {
			klog.Error("Get version by hash failed.", "hash", common.ToHex(datas.StateHash))
			return values
		}

		klog.Debug("KVMVCCStore Get mvcc GetVersion", "hash", common.HashHex(datas.StateHash), "version", version)

		for i := 0; i < len(datas.Keys); i++ {
			value, err := mvccs.mvcc.GetV(datas.Keys[i], version)
			if err != nil {
				klog.Error("GetV by Keys failed.", "Key", string(datas.Keys[i]), "version", version)
			} else if value != nil {
				values[i] = value
				klog.Debug("KVMVCCStore Get mvcc GetV", "index", i, "key", string(datas.Keys[i]), "value", string(value))
			}
		}
	}
	return values
}

func (mvccs *KVMVCCStore) MemSet(datas *types.StoreSet, sync bool) []byte {
	kvset, err := mvccs.checkMVCCFlag(mvccs.GetDB(), datas.Height)
	if err != nil {
		panic(err)
	}

	klog.Debug("KVMVCCStore MemSet checkMVCCFlag", "checkMVCCFlag", kvset, "kvset len", len(kvset))
	if len(kvset) > 0 {
		mvccs.saveKVSets(kvset)
	}

	hash := calcHash(datas)
	klog.Debug("KVMVCCStore MemSet AddMVCC", "prestatehash", common.ToHex(datas.StateHash), "hash", common.ToHex(hash), "height", datas.Height)
	kvlist, err := mvccs.mvcc.AddMVCC(datas.KV, hash, datas.StateHash, datas.Height)
	if err != nil {
		panic(err)
	}
	/*
		for i := 0; i < len(kvlist); i++ {
			klog.Info("KVMVCCStore MemSet AddMVCC", "index", i, "key", string(kvlist[i].Key),"value", string(kvlist[i].Value))
		}*/
	mvccs.kvsetmap[string(hash)] = kvlist

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
	klog.Debug("KVMVCCStore Commit saveKVSets", "hash", common.ToHex(req.Hash))
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

func (mvccs *KVMVCCStore) save(kvmap map[string]*types.KeyValue) {
	storeBatch := mvccs.GetDB().NewBatch(true)
	for _, kv := range kvmap {
		if kv.Value == nil {
			storeBatch.Delete(kv.Key)
		} else {
			storeBatch.Set(kv.Key, kv.Value)
		}
	}
	storeBatch.Write()
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

func (mvccs *KVMVCCStore) checkMVCCFlag(db dbm.DB, height int64) ([]*types.KeyValue, error) {
	//flag = 0 : init
	//flag = 1 : start from zero
	//flag = 2 : start from no zero //不允许flag = 2的情况
	if atomic.LoadInt64(&mvccs.flagMVCC) == FlagInit {
		flag, err := loadFlag(db, types.FlagKeyMVCC)
		if err != nil {
			panic(err)
		}
		atomic.StoreInt64(&mvccs.flagMVCC, flag)
	}
	var kvset []*types.KeyValue
	if atomic.LoadInt64(&mvccs.flagMVCC) == FlagInit {
		if height != 0 {
			atomic.StoreInt64(&mvccs.flagMVCC, FlagNotFromZero)
		} else {
			//区块为0, 写入标志
			if atomic.CompareAndSwapInt64(&mvccs.flagMVCC, FlagInit, FlagFromZero) {
				kvset = append(kvset, types.FlagKV(types.FlagKeyMVCC, FlagFromZero))
			}
		}
	}
	if atomic.LoadInt64(&mvccs.flagMVCC) != FlagFromZero {
		panic("config set enableMVCC=true, it must be synchronized from 0 height")
	}
	return kvset, nil
}

func calcHash(datas proto.Message) []byte {
	b := types.Encode(datas)
	return common.Sha256(b)
}

func loadFlag(db dbm.DB, key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := db.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound || err == dbm.ErrNotFoundInDb {
		return 0, nil
	}
	return 0, err
}
