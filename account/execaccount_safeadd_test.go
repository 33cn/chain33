// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"math"
	"testing"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/db"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func newExecSafeAddCoinsDB(t *testing.T) *DB {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	acc := NewCoinsAccount(cfg)
	memdb, err := db.NewGoMemDB("gomemdb", "exec-safeadd", 128)
	require.NoError(t, err)
	acc.SetDB(memdb)
	return acc
}

// Regression test: exec account balance additions must go through safeAdd,
// same as depositBalance/Mint on the main ledger. An overflowing ExecDeposit
// must be rejected and leave the balance untouched instead of wrapping to
// negative.
func TestExecDepositOverflowRejected(t *testing.T) {
	acc := newExecSafeAddCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	// seed an exec account balance close to the int64 limit
	seed := int64(math.MaxInt64 - 5e15)
	acc.SaveExecAccount(execaddr, &types.Account{Addr: addr1, Balance: seed})

	amount := int64(1e16) // passes CheckAmount (< MaxCoin*1e8 = 1e17)
	_, err := acc.ExecDeposit(addr1, execaddr, amount)
	require.Equal(t, types.ErrAmount, err, "overflowing exec deposit must be rejected")
	require.Equal(t, seed, acc.LoadExecAccount(addr1, execaddr).Balance,
		"balance must be unchanged after rejected overflow deposit")
}

// ExecActive moves frozen coins back to balance; the balance addition must
// also be overflow protected.
func TestExecActiveOverflowRejected(t *testing.T) {
	acc := newExecSafeAddCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	seed := int64(math.MaxInt64 - 5e15)
	frozen := int64(1e16)
	acc.SaveExecAccount(execaddr, &types.Account{Addr: addr1, Balance: seed, Frozen: frozen})

	_, err := acc.ExecActive(addr1, execaddr, frozen)
	require.Equal(t, types.ErrAmount, err, "overflowing exec active must be rejected")
	final := acc.LoadExecAccount(addr1, execaddr)
	require.Equal(t, seed, final.Balance)
	require.Equal(t, frozen, final.Frozen)
}

// execDepositFrozen adds to the frozen balance; it must be overflow protected
// as well.
func TestExecDepositFrozenOverflowRejected(t *testing.T) {
	acc := newExecSafeAddCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	seed := int64(math.MaxInt64 - 5e15)
	acc.SaveExecAccount(execaddr, &types.Account{Addr: addr1, Frozen: seed})

	amount := int64(1e16)
	_, err := acc.execDepositFrozen(addr1, execaddr, amount)
	require.Equal(t, types.ErrAmount, err, "overflowing exec deposit frozen must be rejected")
	require.Equal(t, seed, acc.LoadExecAccount(addr1, execaddr).Frozen)
}

// The MaxTokenBalance cap inside safeAdd must not break normal deposit,
// frozen/active cycles (checkBalance moves coins between frozen and active).
func TestExecAccountNormalPathsWithSafeAdd(t *testing.T) {
	acc := newExecSafeAddCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	// max single amount allowed by CheckAmount is just below 1e17
	amount := int64(1e16)

	// deposit up to exactly MaxTokenBalance is allowed
	acc.SaveExecAccount(execaddr, &types.Account{Addr: addr1, Balance: types.MaxTokenBalance - amount})
	_, err := acc.ExecDeposit(addr1, execaddr, amount)
	require.NoError(t, err)
	require.Equal(t, types.MaxTokenBalance, acc.LoadExecAccount(addr1, execaddr).Balance)

	// freeze, then activate back at the cap: total stays at MaxTokenBalance
	_, err = acc.ExecFrozen(addr1, execaddr, amount)
	require.NoError(t, err)
	_, err = acc.ExecActive(addr1, execaddr, amount)
	require.NoError(t, err)
	final := acc.LoadExecAccount(addr1, execaddr)
	require.Equal(t, types.MaxTokenBalance, final.Balance)
	require.Equal(t, int64(0), final.Frozen)

	// execDepositFrozen up to MaxTokenBalance frozen is allowed
	acc.SaveExecAccount(execaddr, &types.Account{Addr: addr2, Frozen: types.MaxTokenBalance - amount})
	_, err = acc.execDepositFrozen(addr2, execaddr, amount)
	require.NoError(t, err)
	require.Equal(t, types.MaxTokenBalance, acc.LoadExecAccount(addr2, execaddr).Frozen)

	// any further deposit beyond MaxTokenBalance is rejected
	_, err = acc.ExecDeposit(addr1, execaddr, 1)
	require.Equal(t, types.ErrAmount, err)
	require.Equal(t, types.MaxTokenBalance, acc.LoadExecAccount(addr1, execaddr).Balance)
}
