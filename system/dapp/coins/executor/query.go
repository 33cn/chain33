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

// Query_GetTxsByAddr query get txs by address
func (c *Coins) Query_GetTxsByAddr(in *types.ReqAddr) (types.Message, error) {
	return c.GetTxsByAddr(in)
}

// Query_GetPrefixCount query key counts in the prefix
func (c *Coins) Query_GetPrefixCount(in *types.ReqKey) (types.Message, error) {
	return c.GetPrefixCount(in)
}

// Query_GetAddrTxsCount query count of  txs in the address
func (c *Coins) Query_GetAddrTxsCount(in *types.ReqKey) (types.Message, error) {
	return c.GetAddrTxsCount(in)
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
