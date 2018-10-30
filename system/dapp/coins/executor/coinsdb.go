package executor

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：
EventTransfer -> 转移资产
*/

//package none execer for unknow execer
//all none transaction exec ok, execept nofee
//nofee transaction will not pack into block

import (
	"fmt"

	dbm "gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/types"
)

//存储地址上收币的信息
func calcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("LODB-coins-Addr:%s", addr))
}

func geAddrReciverKV(addr string, reciverAmount int64) *types.KeyValue {
	reciver := &types.Int64{reciverAmount}
	amountbytes := types.Encode(reciver)
	kv := &types.KeyValue{calcAddrKey(addr), amountbytes}
	return kv
}

func getAddrReciver(db dbm.KVDB, addr string) (int64, error) {
	reciver := types.Int64{}
	addrReciver, err := db.Get(calcAddrKey(addr))
	if err != nil && err != types.ErrNotFound {
		return 0, err
	}
	if len(addrReciver) == 0 {
		return 0, nil
	}
	err = types.Decode(addrReciver, &reciver)
	if err != nil {
		return 0, err
	}
	return reciver.Data, nil
}

func setAddrReciver(db dbm.KVDB, addr string, reciverAmount int64) error {
	kv := geAddrReciverKV(addr, reciverAmount)
	return db.Set(kv.Key, kv.Value)
}

func updateAddrReciver(cachedb dbm.KVDB, addr string, amount int64, isadd bool) (*types.KeyValue, error) {
	recv, err := getAddrReciver(cachedb, addr)
	if err != nil && err != types.ErrNotFound {
		return nil, err
	}
	if isadd {
		recv += amount
	} else {
		recv -= amount
	}
	setAddrReciver(cachedb, addr, recv)
	//keyvalue
	return geAddrReciverKV(addr, recv), nil
}
