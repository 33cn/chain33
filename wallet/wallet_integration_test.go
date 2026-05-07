//go:build integration

package wallet_test

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestWalletBalance(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(1)
	assert.Nil(t, err)

	header, _ := mocker.GetAPI().GetLastHeader()
	acc := mocker.GetAccount(header.GetStateHash(), mocker.GetGenesisAddress())
	assert.NotNil(t, acc)
	assert.NotEmpty(t, acc.GetAddr())
}

func TestWalletImportKey(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	// Query account list
	req := &types.ReqAccountList{WithoutBalance: true}
	reply, err := api.ExecWalletFunc("wallet", "WalletGetAccountList", req)
	assert.Nil(t, err)
	assert.NotNil(t, reply)
}

func TestWalletUnlockAndLock(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	// Lock
	_, err := api.ExecWalletFunc("wallet", "WalletLock", &types.ReqNil{})
	assert.Nil(t, err)
	// Unlock
	reply, err := api.ExecWalletFunc("wallet", "WalletUnLock", &types.WalletUnLock{Passwd: "123456fuzamei"})
	assert.Nil(t, err)
	assert.NotNil(t, reply)
}

func TestWalletGetTxList(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	err = mocker.WaitHeight(2)
	assert.Nil(t, err)

	api := mocker.GetAPI()
	txs, err := api.GetTxListByAddr(&types.ReqAddrs{
		Addrs: []string{mocker.GetGenesisAddress()},
	})
	assert.Nil(t, err)
	assert.NotNil(t, txs)
}

func TestWalletCheckAddr(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	hotAddr := mocker.GetHotAddress()
	genesisAddr := mocker.GetGenesisAddress()
	assert.NotEmpty(t, hotAddr)
	assert.NotEmpty(t, genesisAddr)
	assert.NotEqual(t, hotAddr, genesisAddr)
}
