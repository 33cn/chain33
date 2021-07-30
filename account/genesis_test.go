// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"testing"

	"github.com/33cn/chain33/types"

	//"fmt"
	"github.com/33cn/chain33/common/address"
	"github.com/stretchr/testify/require"
)

func TestGenesisInit(t *testing.T) {
	accCoin, _ := GenerAccDb()
	accCoin.GenerAccData()
	_, err := accCoin.GenesisInit(addr1, 100*types.DefaultCoinPrecision)
	require.NoError(t, err)
	//t.Logf("GenesisInit is %v", recp)
	t.Logf("GenesisInit [%d]",
		accCoin.LoadAccount(addr1).Balance)
}

func TestGenesisInitExec(t *testing.T) {
	accCoin, _ := GenerAccDb()
	execaddr := address.ExecAddress("coins")
	_, err := accCoin.GenesisInitExec(addr1, 10*types.DefaultCoinPrecision, execaddr)
	require.NoError(t, err)
	//t.Logf("GenesisInitExec Receipt is %v", Receipt)
	t.Logf("GenesisInitExec [%d]___[%d]",
		accCoin.LoadExecAccount(addr1, execaddr).Balance,
		accCoin.LoadAccount(execaddr).Balance)
	require.Equal(t, int64(10*types.DefaultCoinPrecision), accCoin.LoadExecAccount(addr1, execaddr).Balance)
	require.Equal(t, int64(10*types.DefaultCoinPrecision), accCoin.LoadAccount(execaddr).Balance)
}
