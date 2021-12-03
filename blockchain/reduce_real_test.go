// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"testing"

	"github.com/33cn/chain33/blockchain"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/require"
)

func TestTryReduceLocalDB(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mock33 := testnode.NewWithConfig(cfg, nil)
	//发送交易
	chain := mock33.GetBlockChain()
	db := chain.GetDB()
	kvs := getAllKeys(db)
	require.Equal(t, len(kvs), kvCount)
	defer mock33.Close()

	blockchain.ReduceHeight = 0
	defer func() {
		blockchain.ReduceHeight = 10000
	}()

	var flagHeight int64
	count := 10
	for i := 0; i < 2; i++ {
		txs := util.GenCoinsTxs(cfg, mock33.GetGenesisKey(), int64(count))
		for j := 0; j < len(txs); j++ {
			reply, err := mock33.GetAPI().SendTx(txs[j])
			require.Nil(t, err)
			require.Equal(t, reply.IsOk, true)
			waitH := i*count + (j + 1)
			mock33.WaitHeight(int64(waitH))
		}
		flagHeight = chain.TryReduceLocalDB(flagHeight, int64(count))
		require.Equal(t, flagHeight, int64((i+1)*count+1))
	}
}
