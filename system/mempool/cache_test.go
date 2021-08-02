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

func Test_delDelayTx(t *testing.T) {

	cache := newDelayTxCache(100)
	addFunc := func(payload string, delay int) {
		err := cache.addDelayTx(&types.DelayTx{
			Tx:           &types.Transaction{Payload: []byte(payload)},
			EndDelayTime: int64(delay),
		})
		require.Nilf(t, err, "payload:%s, delay:%d", payload, delay)
	}
	//add delay tx
	for i := 0; i < 10; i++ {
		addFunc(fmt.Sprintf("height%d", i), i)
		addFunc(fmt.Sprintf("time%d", i), int(types.ExpireBound)+1+i)
	}
	//no match tx
	txList := cache.delExpiredTxs(0, 0, -1)
	require.Equal(t, 0, len(txList))
	//del delay block height 10
	txList = cache.delExpiredTxs(0, 0, 0)
	require.Equal(t, 1, len(txList))
	require.Equal(t, []byte("height0"), txList[0].Payload)
	//del delay time <= expireBound+5, height=1
	txList = cache.delExpiredTxs(0, types.ExpireBound+5, 1)
	require.Equal(t, 6, len(txList))
	require.Equal(t, []byte("time0"), txList[0].Payload)
	require.Equal(t, []byte("height1"), txList[5].Payload)

	//del delay block time 13 14, block height 19
	txList = cache.delExpiredTxs(0, types.ExpireBound+100, 0)
	require.Equal(t, 5, len(txList))
	//check delay height=2, 5 tx
	_, exist := cache.contains((&types.Transaction{Payload: []byte("height1")}).Hash())
	require.False(t, exist)
	delayTime, exist := cache.contains((&types.Transaction{Payload: []byte("height5")}).Hash())
	require.True(t, exist)
	require.Equal(t, int64(5), delayTime)
}
