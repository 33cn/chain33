// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package plugin

import (
	"fmt"
	"sync"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

// FlagKey gen flag key
func FlagKey(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-Flag", types.LocalPluginPrefix, name))
}

// Flag check plugin enable for height zero
type Flag struct {
	flag int64
	mu   sync.Mutex
}

// CheckFlag enable from height zero
func (f *Flag) CheckFlag(plugin Plugin, flagKey []byte, enable bool) (kvset []*types.KeyValue, ok bool, err error) {
	if !enable {
		return nil, false, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.flag == 0 {
		flag, err := loadFlag(plugin.GetLocalDB(), flagKey)
		if err != nil {
			return nil, false, err
		}
		f.flag = flag
	}
	if plugin.GetHeight() != 0 && f.flag == 0 {
		return nil, false, types.ErrDBFlag
	}
	if plugin.GetHeight() == 0 {
		f.flag = 1
		kvset = append(kvset, types.FlagKV(flagKey, f.flag))
	}
	return kvset, true, nil
}

func loadFlag(localDB dbm.KVDB, key []byte) (int64, error) {
	flag := &types.Int64{}
	flagBytes, err := localDB.Get(key)
	if err == nil {
		err = types.Decode(flagBytes, flag)
		if err != nil {
			return 0, err
		}
		return flag.GetData(), nil
	} else if err == types.ErrNotFound {
		return 0, nil
	}
	return 0, err
}

// UpgradeFlag Upgrade Flag
func UpgradeFlag(localdb dbm.KVDB, name string, oldKey []byte) (enable bool, err error) {
	v, err := localdb.Get(oldKey)
	if err != nil {
		if err == types.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	key := FlagKey(name)
	localdb.Set(oldKey, nil)
	localdb.Set(key, v)
	return true, nil
}
