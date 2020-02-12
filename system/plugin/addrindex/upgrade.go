package addrindex

import (
	"bytes"
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

/*
types.CalcAddrTxsCountKey(addr)
types.CalcTxAddrDirHashKey
types.CalcTxAddrHashKey
*/

// CalcAddrTxsCountPrefixOld 获得老的前缀
func CalcAddrTxsCountPrefixOld() []byte {
	return types.AddrTxsCount
}

// CalcTxAddrHashPrefixOld 获得老的前缀
func CalcTxAddrHashPrefixOld() []byte {
	return types.TxAddrHash
}

// CalcTxAddrDirHashPrefixOld 获得老的前缀
func CalcTxAddrDirHashPrefixOld() []byte {
	return types.TxAddrDirHash
}

// CalcTxAddrHashPrefix 用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
func CalcTxAddrHashPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, TxAddrHash))
}

// CalcTxAddrDirHashPrefix 用于存储地址相关的hash列表，key=TxAddrHash:addr:flag:height*100000 + index
func CalcTxAddrDirHashPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, TxAddrDirHash))
}

// CalcAddrTxsCountPrefix 存储地址参与的交易数量。add时加一，del时减一
func CalcAddrTxsCountPrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, AddrTxsCount))
}

// Upgrade TODO 数量多, 需要测试不分批是否可以
func (p *addrindexPlugin) Upgrade() error {
	toVersion := 2
	elog.Info("Upgrade start", "to_version", toVersion, "plugin", name)
	version, err := plugins.GetVersion(p.GetLocalDB(), name)
	if err != nil {
		return errors.Wrap(err, "Upgrade get version")
	}
	if version >= toVersion {
		elog.Debug("Upgrade not need to upgrade", "current_version", version, "to_version", toVersion)
		return nil
	}
	prefixes := []struct {
		from []byte
		to   []byte
	}{
		{CalcAddrTxsCountPrefixOld(), CalcAddrTxsCountPrefix(name)},
		{CalcTxAddrDirHashPrefixOld(), CalcTxAddrDirHashPrefix(name)},
		{CalcAddrTxsCountPrefixOld(), CalcAddrTxsCountPrefix(name)},
	}

	for _, prefix := range prefixes {
		err := upgradeOneKey(p.GetLocalDB(), prefix.from, prefix.to)
		if err != nil {
			return err
		}
	}

	err = plugins.SetVersion(p.GetLocalDB(), name, toVersion)
	if err != nil {
		return errors.Wrap(err, "Upgrade setVersion")
	}

	elog.Info("Upgrade upgrade done")
	return nil
}

func upgradeOneKey(kvdb dbm.KVDB, oldPrefix, newPrefix []byte) (err error) {
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
