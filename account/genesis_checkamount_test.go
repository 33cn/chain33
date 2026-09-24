// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

// Regression test: GenesisInit must not initialize an account with a negative balance.
func TestGenesisInitNegativeAmountRejected(t *testing.T) {
	accCoin, _ := GenerAccDb()

	receipt, err := accCoin.GenesisInit(addr1, -1000)
	require.Equal(t, types.ErrAmount, err, "negative genesis amount must be rejected")
	require.Nil(t, receipt)
	require.Equal(t, int64(0), accCoin.LoadAccount(addr1).Balance,
		"account must not be created with a negative balance")

	// 0 is accepted: it cannot produce a negative balance, and the code that produced
	// the blocks already on the chain accepted it, so rejecting it would leave those
	// blocks unreplayable. See the comment in GenesisInit.
	receipt, err = accCoin.GenesisInit(addr1, 0)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, int64(0), accCoin.LoadAccount(addr1).Balance)

	// a valid positive genesis amount still works
	receipt, err = accCoin.GenesisInit(addr1, 100*types.DefaultCoinPrecision)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, 100*types.DefaultCoinPrecision, accCoin.LoadAccount(addr1).Balance)
}

// Regression test: the upper bound has to stay where safeAdd puts it (MaxTokenBalance),
// rather than the tighter MaxCoin*coinPrecision that CheckAmount enforces. The token
// executor calls GenesisInit from tokenFinishCreate with the token's total, which the
// pre-create step validated against MaxTokenBalance, so a token already created on a
// chain can legitimately have a total between the two bounds -- rejecting it here makes
// the block that created it unreplayable.
func TestGenesisInitAcceptsAmountBetweenMaxCoinAndMaxTokenBalance(t *testing.T) {
	accCoin, _ := GenerAccDb()

	amount := types.MaxCoin * types.DefaultCoinPrecision
	require.False(t, accCoin.CheckAmount(amount), "precondition: CheckAmount rejects this amount")

	receipt, err := accCoin.GenesisInit(addr1, amount)
	require.NoError(t, err, "an amount between MaxCoin*coinPrecision and MaxTokenBalance must be accepted")
	require.NotNil(t, receipt)
	require.Equal(t, amount, accCoin.LoadAccount(addr1).Balance)

	// the bound safeAdd enforces is still enforced
	_, err = accCoin.GenesisInit(addr1, types.MaxTokenBalance+1)
	require.Equal(t, types.ErrAmount, err, "an amount above MaxTokenBalance must still be rejected")
}
