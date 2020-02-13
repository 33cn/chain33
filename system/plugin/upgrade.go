package plugin

import (
	"bytes"
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

var (
	versionPrefix = "version"
	elog          = log.New("module", "system/plugin")
)

// GetVersion plugin localdb Version
func GetVersion(kvdb dbm.KV, name string) (int, error) {
	value, err := kvdb.Get(versionKey(name))
	if err != nil && err != types.ErrNotFound {
		return 1, err
	}
	if err == types.ErrNotFound {
		return 1, nil
	}
	var v types.Int32
	err = types.Decode(value, &v)
	if err != nil {
		return 1, err
	}
	return int(v.Data), nil
}

// SetVersion set plugin db version
func SetVersion(kvdb dbm.KV, name string, version int) error {
	v := types.Int32{Data: int32(version)}
	x := types.Encode(&v)
	return kvdb.Set(versionKey(name), x)
}

func versionKey(name string) []byte {
	return append(types.LocalPluginPrefix, []byte(fmt.Sprintf("%s-%s", name, versionPrefix))...)
}

// UpgradeOneKey 更新 key 的命名: 变前缀, value不变
func UpgradeOneKey(kvdb dbm.KVDB, oldPrefix, newPrefix []byte) (err error) {
	kvs, err := kvdb.List(oldPrefix, nil, 0, dbm.ListASC|dbm.ListWithKey)
	if err != nil {
		if err == types.ErrNotFound {
			return nil
		}
		return errors.Wrapf(err, "upgradeOneKey list %s", oldPrefix)
	}
	elog.Info("upgradeOneKey", "count", len(kvs), "prefix", string(oldPrefix), "to", string(newPrefix))
	for _, kv := range kvs {
		var kv2 types.KeyValue
		err = types.Decode(kv, &kv2)
		if err != nil {
			return errors.Wrap(err, "upgradeOneKey Decode")
		}

		key := kv2.Key
		newKey := genNewKey(key, oldPrefix, newPrefix)

		kvdb.Set(key, nil)
		kvdb.Set(newKey, kv2.Value)
	}
	return nil
}

func genNewKey(key, prefix1, prefix2 []byte) []byte {
	return bytes.Replace(key, prefix1, prefix2, 1)
}
