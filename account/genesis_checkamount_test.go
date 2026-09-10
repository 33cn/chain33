// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

// Regression test: GenesisInit must validate the amount via CheckAmount and
// reject non-positive amounts instead of initializing a negative balance.
func TestGenesisInitNegativeAmountRejected(t *testing.T) {
	accCoin, _ := GenerAccDb()

	receipt, err := accCoin.GenesisInit(addr1, -1000)
	require.Equal(t, types.ErrAmount, err, "negative genesis amount must be rejected")
	require.Nil(t, receipt)
	require.Equal(t, int64(0), accCoin.LoadAccount(addr1).Balance,
		"account must not be created with a negative balance")

	receipt, err = accCoin.GenesisInit(addr1, 0)
	require.Equal(t, types.ErrAmount, err, "zero genesis amount must be rejected")
	require.Nil(t, receipt)
	require.Equal(t, int64(0), accCoin.LoadAccount(addr1).Balance)

	// a valid positive genesis amount still works
	receipt, err = accCoin.GenesisInit(addr1, 100*types.DefaultCoinPrecision)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, 100*types.DefaultCoinPrecision, accCoin.LoadAccount(addr1).Balance)
}
