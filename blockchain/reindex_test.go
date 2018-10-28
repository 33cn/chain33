package blockchain_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/version"
	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/util"
	"gitlab.33.cn/chain33/chain33/util/testnode"
)

func TestReindex(t *testing.T) {
	mock33 := testnode.New("--notset--", nil)
	//发送交易
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	assert.Equal(t, len(kvs), 19)
	defer mock33.Close()
	txs := util.GenCoinsTxs(mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(1)
	txs = util.GenCoinsTxs(mock33.GetGenesisKey(), 10)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(2)
	txs = util.GenNoneTxs(1)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(3)
	txs = util.GenNoneTxs(2)
	for i := 0; i < len(txs); i++ {
		reply, err := mock33.GetAPI().SendTx(txs[i])
		assert.Nil(t, err)
		assert.Equal(t, reply.IsOk, true)
	}
	mock33.WaitHeight(4)
	kvs1 := getAllKeys(db)
	version.SetLocalDBVersion("10000.0.0")
	chain.UpgradeChain()
	kvs2 := getAllKeys(db)
	assert.Equal(t, kvs1, kvs2)
}

func getAllKeys(db dbm.DB) (kvs []*types.KeyValue) {
	it := db.Iterator(nil, types.EmptyValue, false)
	defer it.Close()
	for it.Rewind(); it.Valid(); it.Next() {
		key := copyBytes(it.Key())
		val := it.ValueCopy()
		//meta 信息是唯一不同的地方
		if string(key) == "LocalDBMeta" {
			continue
		}
		if bytes.HasPrefix(key, []byte("TotalFee")) {
			//println("--", string(key)[0:4], common.ToHex(key))
			totalFee := &types.TotalFee{}
			types.Decode(val, totalFee)
			//println("val", totalFee.String())
		}
		kvs = append(kvs, &types.KeyValue{Key: key, Value: val})
	}
	return kvs
}

func str(key string) string {
	return strings.Replace(key, "\n", "\\n", -1)
}

func copyBytes(keys []byte) []byte {
	data := make([]byte, len(keys))
	copy(data, keys)
	return data
}
