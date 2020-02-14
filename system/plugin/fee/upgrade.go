// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fee

import (
	"fmt"

	"github.com/33cn/chain33/system/plugin"
	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
	"github.com/pkg/errors"
)

// CalcTotalFeePrefixOld 获得老的前缀
func CalcTotalFeePrefixOld() []byte {
	return []byte("TotalFeeKey:")
}

// CalcTotalFeePrefix 用于存储地址相关的hash列表，key=TxAddrHash:addr:height*100000 + index
func CalcTotalFeePrefix(name string) []byte {
	return []byte(fmt.Sprintf("%s-%s-%s:", types.LocalPluginPrefix, name, "Fee"))
}

func (p *feePlugin) Upgrade() error {
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
		{CalcTotalFeePrefixOld(), CalcTotalFeePrefix(name)},
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
