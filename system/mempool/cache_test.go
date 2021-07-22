// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mempool

import (
	"fmt"
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func Test_addDelayTx(t *testing.T) {

	cache := newDelayTxCache(0)
	tx := &types.DelayTx{}

	err := cache.addDelayTx(tx)
	require.Equal(t, types.ErrNilTransaction, err)
	tx.Tx = &types.Transaction{}
	err = cache.addDelayTx(tx)
	require.Equal(t, types.ErrCacheOverFlow, err)
	cache.size = 10
	err = cache.addDelayTx(tx)
	require.Nil(t, err)
	err = cache.addDelayTx(tx)
	require.Equal(t, types.ErrDupTx, err)
	delayTime, exist := cache.contains(tx.Tx.Hash())
	require.True(t, exist)
	require.Equal(t, tx.EndDelayTime, delayTime)
}

func addDelayTxTest(c *delayTxCache, tx *types.Transaction, delayTime int64) error {
	return c.addDelayTx(&types.DelayTx{Tx: tx, EndDelayTime: delayTime})
}

func Test_delDelayTx(t *testing.T) {

	cache := newDelayTxCache(100)
	//add delay time [10, 19]
	for i := 0; i < 10; i++ {
		err := addDelayTxTest(cache, &types.Transaction{Payload: []byte(fmt.Sprintf("test%d", i))}, int64(10+i))
		require.Nil(t, err)
	}
	_ = addDelayTxTest(cache, &types.Transaction{Payload: []byte("last")}, 19)
	//no match tx
	txList := cache.delExpiredTxs(1, 2, 3)
	require.Equal(t, 0, len(txList))
	//del delay block height 10
	txList = cache.delExpiredTxs(1, 2, 10)
	require.Equal(t, 1, len(txList))
	require.Equal(t, []byte("test0"), txList[0].Payload)
	//del delay block time 11 12
	txList = cache.delExpiredTxs(1, 12, 100)
	require.Equal(t, 2, len(txList))
	require.Equal(t, []byte("test1"), txList[0].Payload)

	//del delay block time 13 14, block height 19
	txList = cache.delExpiredTxs(1, 14, 19)
	require.Equal(t, 4, len(txList))
	require.Equal(t, []byte("test3"), txList[0].Payload)
	require.Equal(t, []byte("test4"), txList[1].Payload)
	require.Equal(t, []byte("test9"), txList[2].Payload)
	require.Equal(t, []byte("last"), txList[3].Payload)
	//check delay time= 15 tx
	delayTime, exist := cache.contains((&types.Transaction{Payload: []byte("test5")}).Hash())
	require.True(t, exist)
	require.Equal(t, int64(15), delayTime)
}
