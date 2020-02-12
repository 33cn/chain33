package addrindex

import (
	"testing"

	plugins "github.com/33cn/chain33/system/plugin"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/stretchr/testify/assert"
)

func TestGenNewKey(t *testing.T) {
	old := []byte("old:1223")
	x := genNewKey(old, []byte("old:"), []byte("new:"))
	assert.Equal(t, []byte("new:1223"), x)
	assert.Equal(t, []byte("old:1223"), old)
}

func Test_Upgrade(t *testing.T) {
	dir, db, localdb := util.CreateTestDB()
	defer util.CloseTestDB(dir, db)
	assert.NotNil(t, localdb)

	// test empty db
	p := newAddrindex()
	p.SetLocalDB(localdb)
	err := p.Upgrade()
	assert.Nil(t, err)

	// test again
	plugins.SetVersion(localdb, name, 1)
	err = p.Upgrade()
	assert.Nil(t, err)

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
	err = p.Upgrade()
	assert.Nil(t, err)

	// 已经是升级后的版本了， 不需要再升级
	err = p.Upgrade()
	assert.Nil(t, err)

	// 先修改版本去升级，但数据已经升级了， 所以处理数据量为0
	plugins.SetVersion(localdb, name, 1)
	err = p.Upgrade()
	assert.Nil(t, err)

	// just print log
	//assert.NotNil(t, nil)
}
