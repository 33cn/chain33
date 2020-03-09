package addrindex

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
	p := newAddrindex()
	p.SetLocalDB(localdb)
	done, err := p.Upgrade(100)
	assert.Nil(t, err)
	assert.True(t, done)

	// test again
	plugins.SetVersion(localdb, name, 1)
	done, err = p.Upgrade(100)
	assert.Nil(t, err)
	assert.True(t, done)

	// test with data
	addresses := []string{"addr1", "addr2", "addr3"}
	indexes := []string{"0001000001", "0001000002", "0001000003"}
	data := [][]byte{[]byte("xx1"), []byte("xx2"), []byte("xx3")}
	for i := 0; i < 3; i++ {
		localdb.Set(types.CalcTxAddrHashKey(addresses[i], indexes[i]), data[i])
		localdb.Set(types.CalcTxAddrDirHashKey(addresses[i], 1, indexes[i]), data[i])
		localdb.Set(types.CalcAddrTxsCountKey(addresses[i]), data[i])
	}

	plugins.SetVersion(localdb, name, 1)
	done, err = p.Upgrade(2)
	assert.Nil(t, err)
	assert.False(t, done)

	done, err = p.Upgrade(10)
	assert.Nil(t, err)
	assert.True(t, done)

	// 已经是升级后的版本了， 不需要再升级
	done, err = p.Upgrade(2)
	assert.Nil(t, err)
	assert.True(t, done)

	// 先修改版本去升级，但数据已经升级了， 所以处理数据量为0
	plugins.SetVersion(localdb, name, 1)
	done, err = p.Upgrade(2)
	assert.Nil(t, err)
	assert.True(t, done)

	// just print log
	//assert.NotNil(t, nil)
}
