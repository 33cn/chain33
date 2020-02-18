// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kvmvcc

import (
	"fmt"

	log "github.com/33cn/chain33/common/log/log15"
	"github.com/33cn/chain33/system/plugin"
	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

var elog = log.New("module", "system/plugin/mvcc")

// PrefixOld 获得老的前缀
func PrefixOld() []byte {
	return []byte(".-mvcc-.")
}

// Prefix 新前缀
func Prefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-", types.LocalPluginPrefix, name))
}

func (p *mvccPlugin) Upgrade() error {
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

	// 规则不变, 直接换前缀, 不通过各类数据分别升级
	prefixes := []struct {
		from []byte
		to   []byte
	}{
		{PrefixOld(), Prefix(name)},
	}

	for _, prefix := range prefixes {
		err := plugin.UpgradeOneKey(p.GetLocalDB(), prefix.from, prefix.to)
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
