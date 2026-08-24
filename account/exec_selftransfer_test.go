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

// 同一 eth 地址的两种字符串表示（仅大小写不同）。
// address.FormatAddrKey 对 eth 地址统一转小写后作为 DB key，
// 因此二者指向状态库中的同一条账户记录。
const (
	ethAddrMixed = "0x111111111111111111111111111111111111111A"
	ethAddrLower = "0x111111111111111111111111111111111111111a"
)

func newTestCoinsDB(t *testing.T) *DB {
	cfg := types.NewChain33Config(types.GetDefaultCfgstring())
	acc := NewCoinsAccount(cfg)
	memdb, err := db.NewGoMemDB("gomemdb", "self-transfer-test", 128)
	require.NoError(t, err)
	acc.SetDB(memdb)
	return acc
}

// 前置条件：两个不同字符串的 eth 地址格式化后指向同一个 DB key。
func TestEthAddrCaseVariantsSameKey(t *testing.T) {
	require.NotEqual(t, ethAddrMixed, ethAddrLower)
	require.True(t, address.IsEthAddress(ethAddrMixed))
	require.True(t, address.IsEthAddress(ethAddrLower))
	require.Equal(t, address.FormatAddrKey(ethAddrMixed), address.FormatAddrKey(ethAddrLower))
}

// 回归测试：ExecTransfer 自我转账检查必须比较归一化后的地址。
// 传入同一 eth 地址的两种大小写形式，原始字符串比较不相等，
// 但存储层经 FormatAddrKey 归一化后指向同一账户记录，
// 若不拦截会导致先扣款记录被加款记录覆盖，余额凭空增加（造币）。
// 修复后该转账必须被拒绝且余额不变。
func TestExecTransferSelfTransferCaseVariantRejected(t *testing.T) {
	acc := newTestCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	// 播种：该 eth 地址在执行器账户中有 100 coins
	origin := 100 * types.DefaultCoinPrecision
	amount := 10 * types.DefaultCoinPrecision
	acc.SaveExecAccount(execaddr, &types.Account{Addr: ethAddrLower, Balance: origin})

	// from 与 to 是同一地址的不同大小写形式，必须被识别为自我转账
	_, err := acc.ExecTransfer(ethAddrMixed, ethAddrLower, execaddr, amount)
	require.Equal(t, types.ErrSendSameToRecv, err)

	// 两个方向的大小写组合都必须被拒绝
	_, err = acc.ExecTransfer(ethAddrLower, ethAddrMixed, execaddr, amount)
	require.Equal(t, types.ErrSendSameToRecv, err)

	// 余额不变，未发生造币
	require.Equal(t, origin, acc.LoadExecAccount(ethAddrLower, execaddr).Balance)
	require.Equal(t, origin, acc.LoadExecAccount(ethAddrMixed, execaddr).Balance)
}

// 回归测试：修复不影响正常（不同地址）的 ExecTransfer。
func TestExecTransferNormalNotAffected(t *testing.T) {
	acc := newTestCoinsDB(t)
	execaddr := address.ExecAddress("ticket")

	origin := 100 * types.DefaultCoinPrecision
	amount := 10 * types.DefaultCoinPrecision
	acc.SaveExecAccount(execaddr, &types.Account{Addr: addr1, Balance: origin})

	receipt, err := acc.ExecTransfer(addr1, addr2, execaddr, amount)
	require.NoError(t, err)
	require.NotNil(t, receipt)
	require.Equal(t, origin-amount, acc.LoadExecAccount(addr1, execaddr).Balance)
	require.Equal(t, amount, acc.LoadExecAccount(addr2, execaddr).Balance)
}
