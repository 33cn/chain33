// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"testing"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/db"
	_ "github.com/33cn/chain33/system/address" //init address drivers (btc/eth)
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

// same eth address in two string forms, only letter case differs.
// address.FormatAddrKey normalizes eth addresses to lower case as the DB key,
// so both forms point to the same account record in the state db.
const (
	testEthAddrMixed = "0x111111111111111111111111111111111111111A"
	testEthAddrLower = "0x111111111111111111111111111111111111111a"
)

func newTestCoinsDB(t *testing.T) *DB {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	acc := NewCoinsAccount(cfg)
	memdb, err := db.NewGoMemDB("gomemdb", "exec-frozen", 128)
	require.NoError(t, err)
	acc.SetDB(memdb)
	return acc
}

// ExecTransferFrozen must reject self-transfer even when from/to are the same
// address in different letter cases. Before the fix, the raw string comparison
// from == to was bypassed: the from record (Frozen decreased) was overwritten
// by the to record (Balance increased) under the same normalized DB key,
// duplicating frozen funds.
func TestExecTransferFrozenRejectSelfTransferCaseVariant(t *testing.T) {
	acc := newTestCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	originBalance := int64(100 * types.DefaultCoinPrecision)
	originFrozen := int64(20 * types.DefaultCoinPrecision)
	amount := int64(10 * types.DefaultCoinPrecision)
	acc.SaveExecAccount(execaddr, &types.Account{
		Addr: testEthAddrLower, Balance: originBalance, Frozen: originFrozen,
	})

	_, err := acc.ExecTransferFrozen(testEthAddrMixed, testEthAddrLower, execaddr, amount)
	require.Equal(t, types.ErrSendSameToRecv, err,
		"self-transfer across address case variants must be rejected")

	final := acc.LoadExecAccount(testEthAddrLower, execaddr)
	require.Equal(t, originBalance, final.Balance, "Balance must be unchanged")
	require.Equal(t, originFrozen, final.Frozen, "Frozen must be unchanged")
}

// ExecTransferFrozen between two different addresses must still work normally.
func TestExecTransferFrozenNormal(t *testing.T) {
	acc := newTestCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	addrFrom := "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	addrTo := "12qyocayNF7Lv6C9qW4avxs2E7U41fKSfv"
	originBalance := int64(100 * types.DefaultCoinPrecision)
	originFrozen := int64(20 * types.DefaultCoinPrecision)
	amount := int64(10 * types.DefaultCoinPrecision)
	acc.SaveExecAccount(execaddr, &types.Account{
		Addr: addrFrom, Balance: originBalance, Frozen: originFrozen,
	})

	_, err := acc.ExecTransferFrozen(addrFrom, addrTo, execaddr, amount)
	require.NoError(t, err)

	from := acc.LoadExecAccount(addrFrom, execaddr)
	require.Equal(t, originBalance, from.Balance)
	require.Equal(t, originFrozen-amount, from.Frozen)
	to := acc.LoadExecAccount(addrTo, execaddr)
	require.Equal(t, amount, to.Balance)
	require.Equal(t, int64(0), to.Frozen)
}
