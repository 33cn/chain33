// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mvcc

import (
	"fmt"

	log "github.com/33cn/chain33/common/log/log15"
	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
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

func (p *mvccPlugin) Upgrade(count int32) (bool, error) {
	toVersion := 2
	prefixes := []plugins.Prefixes{
		{PrefixOld(), Prefix(name)},
	}
	return plugins.Upgrade(p.GetLocalDB(), name, toVersion, prefixes, count)
}
