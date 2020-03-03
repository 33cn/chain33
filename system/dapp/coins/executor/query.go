// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/types"
)

// Query_GetAddrReciver query of get address reciver
func (c *Coins) Query_GetAddrReciver(in *types.ReqAddr) (types.Message, error) {
	return c.GetAddrReciver(in)
}

// Query_GetPrefixCount query key counts in the prefix
func (c *Coins) Query_GetPrefixCount(in *types.ReqKey) (types.Message, error) {
	return c.GetPrefixCount(in)
}

// GetAddrReciver get address reciver by address
func (c *Coins) GetAddrReciver(addr *types.ReqAddr) (types.Message, error) {
	reciver := types.Int64{}
	db := c.GetLocalDB()
	addrReciver, err := db.Get(calcAddrKey(addr.Addr))
	if addrReciver == nil || err != nil {
		return &reciver, types.ErrEmpty
	}
	err = types.Decode(addrReciver, &reciver)
	if err != nil {
		return &reciver, err
	}
	return &reciver, nil
}

// 保证现有调用有效, 保留空函数, 和jrpc的实现有关
// 在现有的接口实现时, 需要看执行器的接口的返回类型, 做json到pb的输出数据转化.

// Query_GetTxsByAddr query get txs by address
func (c *Coins) Query_GetTxsByAddr(in *types.ReqAddr) (types.Message, error) {
	return nil, nil
}

// Query_GetAddrTxsCount query count of  txs in the address
func (c *Coins) Query_GetAddrTxsCount(in *types.ReqKey) (types.Message, error) {
	return nil, nil
}
