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

// Prefixes 前缀
type Prefixes struct {
	From []byte
	To   []byte
}

// Upgrade 升级前缀
func Upgrade(kvdb dbm.KVDB, name string, toVersion int, prefixes []Prefixes, count int32) (bool, error) {
	elog.Info("Upgrade start", "plugin", name, "to_version", toVersion)
	version, err := GetVersion(kvdb, name)
	if err != nil {
		return false, errors.Wrapf(err, "Upgrade get version: %s", name)
	}
	if version >= toVersion {
		elog.Debug("Upgrade not need to upgrade", "current_version", version, "to_version", toVersion)
		return true, nil
	}
	/*
		prefixes := []struct {
			from []byte
			to   []byte
		}{
			{CalcTxPrefixOld(), CalcTxPrefix(name)},
			{CalcTxShortHashPerfixOld(), CalcTxShortPerfix(name)},
		}*/

	for _, prefix := range prefixes {
		done, err := UpgradeOneKey(kvdb, prefix.From, prefix.To, count)
		if err != nil {
			return done, err
		}
		// 未完成, 意味着数据量处理够了, 先返回
		if done == false {
			return done, nil
		}
	}

	err = SetVersion(kvdb, name, toVersion)
	if err != nil {
		return false, errors.Wrapf(err, "Upgrade setVersion: %s", name)
	}

	elog.Info("Upgrade upgrade done")
	return true, nil
}

// UpgradeOneKey 更新 key 的命名: 变前缀, value不变
func UpgradeOneKey(kvdb dbm.KVDB, oldPrefix, newPrefix []byte, count int32) (done bool, err error) {
	kvs, err := kvdb.List(oldPrefix, nil, count, dbm.ListASC|dbm.ListWithKey)
	if err != nil {
		if err == types.ErrNotFound {
			return true, nil
		}
		return false, errors.Wrapf(err, "upgradeOneKey list %s", oldPrefix)
	}
	elog.Info("upgradeOneKey", "count", len(kvs), "prefix", string(oldPrefix), "to", string(newPrefix))
	for _, kv := range kvs {
		var kv2 types.KeyValue
		err = types.Decode(kv, &kv2)
		if err != nil {
			return false, errors.Wrap(err, "upgradeOneKey Decode")
		}

		key := kv2.Key
		newKey := genNewKey(key, oldPrefix, newPrefix)

		kvdb.Set(key, nil)
		kvdb.Set(newKey, kv2.Value)
	}
	return int32(len(kvs)) != count, nil
}

func genNewKey(key, prefix1, prefix2 []byte) []byte {
	return bytes.Replace(key, prefix1, prefix2, 1)
}
