// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

/*
coins 是一个货币的exec。内置货币的执行器。

主要提供两种操作：
EventTransfer -> 转移资产
*/

// package none execer for unknow execer
// all none transaction exec ok, execept nofee
// nofee transaction will not pack into block

import (
	"fmt"

	dbm "github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
)

// calcAddrKey store information on the receiving address
func calcAddrKey(addr string) []byte {
	return []byte(fmt.Sprintf("LODB-coins-Addr:%s", addr))
}

func geAddrReciverKV(addr string, reciverAmount int64) *types.KeyValue {
	reciver := &types.Int64{Data: reciverAmount}
	amountbytes := types.Encode(reciver)
	kv := &types.KeyValue{Key: calcAddrKey(addr), Value: amountbytes}
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
	err = setAddrReciver(cachedb, addr, recv)
	if err != nil {
		return nil, err
	}
	//keyvalue
	return geAddrReciverKV(addr, recv), nil
}
