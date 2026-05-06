package executor_test

import (
	"testing"

	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestExecBlockAndState(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	header, err := api.GetLastHeader()
	assert.Nil(t, err)
	assert.NotNil(t, header.GetStateHash())
	assert.GreaterOrEqual(t, header.Height, int64(0))
}

func TestSendTxAndQuery(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	tx := util.CreateCoinsTx(cfg, mocker.GetGenesisKey(), mocker.GetHotAddress(), 100*types.DefaultCoinPrecision)
	hash := mocker.SendTx(tx)
	assert.NotNil(t, hash)

	err = mocker.Wait()
	assert.Nil(t, err)

	detail, err := mocker.WaitTx(hash)
	assert.Nil(t, err)
	assert.NotNil(t, detail)
}

func TestGetMempoolIntegration(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	txs, err := api.GetMempool(&types.ReqGetMempool{})
	assert.Nil(t, err)
	assert.NotNil(t, txs)
}

func TestGetProperFee(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	api := mocker.GetAPI()
	fee, err := api.GetProperFee(&types.ReqProperFee{TxCount: 1, TxSize: 100})
	assert.Nil(t, err)
	assert.NotNil(t, fee)
	assert.Greater(t, fee.ProperFee, int64(0))
}

func TestGetLastHeader(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	header, err := api.GetLastHeader()
	assert.Nil(t, err)
	assert.NotNil(t, header)
	assert.GreaterOrEqual(t, header.Height, int64(0))
}

func TestGetAddrOverview(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	overview, err := api.GetAddrOverview(&types.ReqAddr{Addr: mocker.GetGenesisAddress()})
	assert.Nil(t, err)
	assert.NotNil(t, overview)
}

func TestGetTransactionByAddr(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	details, err := api.GetTransactionByAddr(&types.ReqAddr{Addr: mocker.GetGenesisAddress(), Count: 5, Direction: 1})
	assert.Nil(t, err)
	assert.NotNil(t, details)
}
