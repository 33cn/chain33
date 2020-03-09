// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stat

import (
	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

func (p *statPlugin) Upgrade(count int32) (bool, error) {
	toVersion := 2
	elog.Info("Upgrade start", "to_version", toVersion, "plugin", name)
	version, err := plugins.GetVersion(p.GetLocalDB(), name)
	if err != nil {
		return false, errors.Wrap(err, "Upgrade get version")
	}
	if version >= toVersion {
		elog.Debug("Upgrade not need to upgrade", "current_version", version, "to_version", toVersion)
		return true, nil
	}

	enable, err := plugins.UpgradeFlag(p.GetLocalDB(), name, types.StatisticFlag())
	if err != nil {
		return false, err
	}
	if enable == false {
		return true, nil
	}

	err = plugins.SetVersion(p.GetLocalDB(), name, toVersion)
	if err != nil {
		return false, errors.Wrap(err, "Upgrade setVersion")
	}

	elog.Info("Upgrade upgrade done")
	return true, nil
}
