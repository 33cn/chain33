// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/types"
)

func (c *Coins) Query_GetAddrReciver(in *types.ReqAddr) (types.Message, error) {
	return c.GetAddrReciver(in)
}

func (c *Coins) Query_GetTxsByAddr(in *types.ReqAddr) (types.Message, error) {
	return c.GetTxsByAddr(in)
}

func (c *Coins) Query_GetPrefixCount(in *types.ReqKey) (types.Message, error) {
	return c.GetPrefixCount(in)
}

func (c *Coins) Query_GetAddrTxsCount(in *types.ReqKey) (types.Message, error) {
	return c.GetAddrTxsCount(in)
}

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
