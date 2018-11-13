// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package account

import (
	"testing"
	//"fmt"

	"github.com/33cn/chain33/common/address"
	//"github.com/33cn/chain33/queue"
	"github.com/33cn/chain33/types"
	"github.com/stretchr/testify/require"
)

func (acc *DB) GenerExecAccData(execaddr string) {
	// 加入账户
	account := &types.Account{
		Balance: 1000 * 1e8,
		Frozen:  20 * 1e8,
		Addr:    addr1,
	}
	acc.SaveExecAccount(execaddr, account)
	acc.SaveAccount(account)

	account.Balance = 900 * 1e8
	account.Frozen = 20 * 1e8
	account.Addr = addr2
	acc.SaveExecAccount(execaddr, account)
	acc.SaveAccount(account)
}

func TestLoadExecAccountQueue(t *testing.T) {

	q := initEnv()
	qAPI, _ := initQueAPI(q)
	blockchainProcess(q)
	storeProcess(q)

	execaddress := address.ExecAddress("ticket")
	accCoin, _ := GenerAccDb()
	acc, err := accCoin.LoadExecAccountQueue(qAPI, addr1, execaddress)
	require.NoError(t, err)
	t.Logf("LoadExecAccountQueue is %v", acc)
}

func TestTransferToExec(t *testing.T) {
	q := initEnv()
	blockchainProcess(q)
	storeProcess(q)

	accCoin, _ := GenerAccDb()
	accCoin.GenerAccData()
	_, err := accCoin.TransferToExec(addr1, addr2, 10*1e8)
	require.NoError(t, err)
	t.Logf("TransferToExec from addr balance [%d] to addr balance [%d]",
		accCoin.LoadAccount(addr1).Balance,
		accCoin.LoadAccount(addr2).Balance)
	require.Equal(t, int64(1000*1e8-10*1e8), accCoin.LoadAccount(addr1).Balance)
	require.Equal(t, int64(900*1e8+10*1e8), accCoin.LoadAccount(addr2).Balance)
}

func TestTransferWithdraw(t *testing.T) {
	q := initEnv()
	blockchainProcess(q)
	storeProcess(q)

	accCoin, _ := GenerAccDb()
	execaddr := address.ExecAddress("coins")

	account := &types.Account{
		Balance: 1000 * 1e8,
		Frozen:  0,
		Addr:    execaddr,
	}
	//往合约地址中打钱
	accCoin.SaveAccount(account)

	//往执行账户中打钱
	account.Addr = addr1
	accCoin.SaveExecAccount(execaddr, account)

	_, err := accCoin.TransferWithdraw(addr1, execaddr, 10*1e8)
	require.NoError(t, err)
	t.Logf("TransferWithdraw [%d]___[%d]",
		accCoin.LoadAccount(execaddr).Balance,
		accCoin.LoadExecAccount(addr1, execaddr).Balance)
}

func TestExecFrozen(t *testing.T) {
	q := initEnv()
	blockchainProcess(q)
	storeProcess(q)

	execaddress := address.ExecAddress("coins")
	accCoin, _ := GenerAccDb()
	accCoin.GenerExecAccData(execaddress)
	_, err := accCoin.ExecFrozen(addr1, execaddress, 10*1e8)
	require.NoError(t, err)

	t.Logf("ExecFrozen [%d]___[%d]",
		accCoin.LoadExecAccount(addr1, execaddress).Balance,
		accCoin.LoadExecAccount(addr1, execaddress).Frozen)

	require.Equal(t, int64(1000*1e8-10*1e8), accCoin.LoadExecAccount(addr1, execaddress).Balance)
	require.Equal(t, int64(20*1e8+10*1e8), accCoin.LoadExecAccount(addr1, execaddress).Frozen)
}

func TestExecActive(t *testing.T) {
	q := initEnv()
	blockchainProcess(q)
	storeProcess(q)

	execaddress := address.ExecAddress("coins")
	accCoin, _ := GenerAccDb()
	accCoin.GenerExecAccData(execaddress)
	_, err := accCoin.ExecActive(addr1, execaddress, 10*1e8)
	require.NoError(t, err)

	t.Logf("ExecActive [%d]___[%d]",
		accCoin.LoadExecAccount(addr1, execaddress).Balance,
		accCoin.LoadExecAccount(addr1, execaddress).Frozen)

	require.Equal(t, int64(1000*1e8+10*1e8), accCoin.LoadExecAccount(addr1, execaddress).Balance)
	require.Equal(t, int64(20*1e8-10*1e8), accCoin.LoadExecAccount(addr1, execaddress).Frozen)
}

func TestExecTransfer(t *testing.T) {
	q := initEnv()
	blockchainProcess(q)
	storeProcess(q)

	execaddress := address.ExecAddress("coins")
	accCoin, _ := GenerAccDb()
	accCoin.GenerExecAccData(execaddress)
	_, err := accCoin.ExecTransfer(addr1, addr2, execaddress, 10*1e8)
	require.NoError(t, err)
	t.Logf("ExecActive [%d]___[%d]",
		accCoin.LoadExecAccount(addr1, execaddress).Balance,
		accCoin.LoadExecAccount(addr2, execaddress).Balance)
	require.Equal(t, int64(1000*1e8-10*1e8), accCoin.LoadExecAccount(addr1, execaddress).Balance)
	require.Equal(t, int64(900*1e8+10*1e8), accCoin.LoadExecAccount(addr2, execaddress).Balance)
}

func TestExecTransferFrozen(t *testing.T) {
	q := initEnv()
	blockchainProcess(q)
	storeProcess(q)

	execaddress := address.ExecAddress("coins")
	accCoin, _ := GenerAccDb()
	accCoin.GenerExecAccData(execaddress)
	_, err := accCoin.ExecTransferFrozen(addr1, addr2, execaddress, 10*1e8)
	require.NoError(t, err)
	t.Logf("TransferFrozen [%d]___[%d]",
		accCoin.LoadExecAccount(addr1, execaddress).Frozen,
		accCoin.LoadExecAccount(addr2, execaddress).Balance)
	require.Equal(t, int64(20*1e8-10*1e8), accCoin.LoadExecAccount(addr1, execaddress).Frozen)
	require.Equal(t, int64(900*1e8+10*1e8), accCoin.LoadExecAccount(addr2, execaddress).Balance)
}

func TestExecDepositFrozen(t *testing.T) {
	q := initEnv()
	blockchainProcess(q)
	storeProcess(q)

	execaddress := address.ExecAddress("ticket")
	accCoin, _ := GenerAccDb()
	accCoin.GenerExecAccData(execaddress)
	_, err := accCoin.ExecDepositFrozen(addr1, execaddress, 25*1e8)
	require.NoError(t, err)
	t.Logf("ExecDepositFrozen [%d]",
		accCoin.LoadExecAccount(addr1, execaddress).Frozen)
	require.Equal(t, int64(20*1e8+25*1e8), accCoin.LoadExecAccount(addr1, execaddress).Frozen)
}
