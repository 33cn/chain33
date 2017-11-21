package account

//package for account manger
//1. load from db
//2. save to db
//3. KVSet
//4. Transfer
//5. Add
//6. Sub
//7. Account balance query
//8. gen a private key -> private key to address (bitcoin likes)

import (
	dbm "code.aliyun.com/chain33/chain33/common/db"
	"code.aliyun.com/chain33/chain33/types"
)

func LoadAccount(db dbm.KVDB, addr []byte) (*types.Account, error) {
	return nil, nil
}

func SaveAccount(db dbm.KVDB, acc *types.Account) {

}

func GetKVSet(acc *types.Account) (kvset []*types.KeyValue) {
	return nil
}
