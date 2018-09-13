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
var maxRollbackNum = 200

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
	kvsetmap map[string][]*types.KeyValue
}

func New(cfg *types.Store) queue.Module {
	bs := drivers.NewBaseStore(cfg)
	kvs := &KVMVCCStore{bs, dbm.NewSimpleMVCC(dbm.NewKVDB(bs.GetDB())), make(map[string][]*types.KeyValue)}
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

	klog.Info("KVMVCCStore Set saveKVSets", "hash", common.ToHex(datas.StateHash), "height", datas.Height)
	mvccs.saveKVSets(kvlist)
	return hash
}

func (mvccs *KVMVCCStore) Get(datas *types.StoreGet) [][]byte {
	values := make([][]byte, len(datas.Keys))

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

	return values
}

func (mvccs *KVMVCCStore) MemSet(datas *types.StoreSet, sync bool) []byte {
	kvset, err := mvccs.checkVersion(datas.Height)
	if err != nil {
		panic(err)
	}

	hash := calcHash(datas)
	klog.Debug("KVMVCCStore MemSet AddMVCC", "prestatehash", common.ToHex(datas.StateHash), "hash", common.ToHex(hash), "height", datas.Height)
	kvlist, err := mvccs.mvcc.AddMVCC(datas.KV, hash, datas.StateHash, datas.Height)
	if err != nil {
		panic(err)
	}

	if len(kvlist) > 0 {
		kvset = append(kvset, kvlist...)
	}

	mvccs.kvsetmap[string(hash)] = kvset

	return hash
}

func (mvccs *KVMVCCStore) Commit(req *types.ReqHash) ([]byte, error) {
	_, ok := mvccs.kvsetmap[string(req.Hash)]
	if !ok {
		klog.Error("store kvmvcc commit", "err", types.ErrHashNotFound)
		return nil, types.ErrHashNotFound
	}
	klog.Debug("KVMVCCStore Commit saveKVSets", "hash", common.ToHex(req.Hash))
	mvccs.saveKVSets(mvccs.kvsetmap[string(req.Hash)])
	delete(mvccs.kvsetmap, string(req.Hash))
	return req.Hash, nil
}

func (mvccs *KVMVCCStore) Rollback(req *types.ReqHash) []byte {
	_, ok := mvccs.kvsetmap[string(req.Hash)]
	if !ok {
		klog.Error("store kvmvcc rollback", "err", types.ErrHashNotFound)
		return nil
	}

	klog.Debug("KVMVCCStore Rollback", "hash", common.ToHex(req.Hash))

	delete(mvccs.kvsetmap, string(req.Hash))
	return req.Hash
}

func (mvccs *KVMVCCStore) IterateRangeByStateHash(statehash []byte, start []byte, end []byte, ascending bool, fn func(key, value []byte) bool) {
	panic("empty")
	//TODO:
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
	maxVersion, err := mvccs.mvcc.GetMaxVersion()
	if err != nil {
		if err != types.ErrNotFound {
			klog.Error("store kvmvcc checkVersion GetMaxVersion failed", "err", err)
			panic(err)
		} else {
			maxVersion = -1
		}
	}

	//zzh
	klog.Debug("store kvmvcc checkVersion ", "maxVersion", maxVersion, "currentVersion", height)

	var kvset []*types.KeyValue
	if maxVersion < height-1 {
		klog.Error("store kvmvcc checkVersion found statehash lost", "maxVersion", maxVersion, "height", height)
		return nil, types.ErrStateHashLost
	} else if maxVersion == height-1 {
		return nil, nil
	} else {
		count := 1
		for i := maxVersion; i >= height; i-- {
			hash, err := mvccs.mvcc.GetVersionHash(i)
			if err != nil {
				klog.Warn("store kvmvcc checkVersion GetVersionHash failed", "height", i, "maxVersion", maxVersion)
				continue
			}
			kvlist, err := mvccs.mvcc.DelMVCC4Height(hash, i)
			if err != nil {
				klog.Warn("store kvmvcc checkVersion DelMVCC failed", "height", i, "err", err)
				continue
			}
			kvset = append(kvset, kvlist...)

			klog.Debug("store kvmvcc checkVersion DelMVCC4Height", "height", i, "maxVersion", maxVersion)
			//为避免高度差过大时出现异常，做一个保护，一次最多回滚200个区块
			count++
			if count >= maxRollbackNum {
				break
			}
		}
	}

	return kvset, nil
}

func calcHash(datas proto.Message) []byte {
	b := types.Encode(datas)
	return common.Sha256(b)
}
