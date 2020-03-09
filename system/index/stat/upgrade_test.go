// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package stat

import (
	"testing"

	plugins "github.com/33cn/chain33/system/index"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func Test_Upgrade(t *testing.T) {
	dir, db, localdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, db)
	assert.NotNil(t, localdb)

	// test empty db
	p := newStat()
	p.SetLocalDB(localdb)
	_, err := p.Upgrade(10)
	assert.Nil(t, err)

	// test again
	plugins.SetVersion(localdb, name, 1)
	_, err = p.Upgrade(10)
	assert.Nil(t, err)

	// test with data
	flagValue := types.Int64{Data: 1}
	v := types.Encode(&flagValue)
	localdb.Set(types.StatisticFlag(), v)

	plugins.SetVersion(localdb, name, 1)
	done, err := p.Upgrade(10)
	assert.Nil(t, err)
	assert.True(t, done)

	// 已经是升级后的版本了， 不需要再升级
	_, err = p.Upgrade(10)
	assert.Nil(t, err)

	// 先修改版本去升级，但数据已经升级了， 所以处理数据量为0
	plugins.SetVersion(localdb, name, 1)
	_, err = p.Upgrade(10)
	assert.Nil(t, err)

	// just print log
	//assert.NotNil(t, nil)
}
