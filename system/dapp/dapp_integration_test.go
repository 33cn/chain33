package dapp_test

import (
	"encoding/hex"
	"testing"

	"github.com/33cn/chain33/common/address"
	_ "github.com/33cn/chain33/system"
	"github.com/33cn/chain33/types"
	"github.com/33cn/chain33/util/testnode"
	"github.com/stretchr/testify/assert"
)

func TestDriverLoadAndExec(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()
	bc := mocker.GetBlockChain()

	// Test chain is running
	assert.NotNil(t, bc)
	assert.NotNil(t, api)

	// Test that drivers are loaded properly
	header, err := api.GetLastHeader()
	assert.Nil(t, err)
	assert.NotNil(t, header)
	assert.GreaterOrEqual(t, header.Height, int64(0))

	// Test get blocks
	blocks, err := api.GetBlocks(&types.ReqBlocks{Start: 0, End: header.Height, IsDetail: false})
	assert.Nil(t, err)
	assert.NotNil(t, blocks)
	assert.GreaterOrEqual(t, len(blocks.Items), 1)
}

func TestDriverExecAddress(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	// Verify exec addresses are computed correctly
	coinsAddr := address.ExecAddress("coins")
	assert.NotEmpty(t, coinsAddr)

	ticketAddr := address.ExecAddress("ticket")
	assert.NotEmpty(t, ticketAddr)
	assert.NotEqual(t, coinsAddr, ticketAddr)
}

func TestGetExecBalance(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	api := mocker.GetAPI()

	// Query genesis account
	genesisAddr := "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	overview, err := api.GetAddrOverview(&types.ReqAddr{Addr: genesisAddr})
	assert.Nil(t, err)
	assert.NotNil(t, overview)
}

func TestFormatTxIntegration(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()

	// Test transaction formatting through the executor
	tx := &types.Transaction{Payload: []byte("test-payload")}
	formatted, err := types.FormatTx(cfg, "coins", tx)
	assert.Nil(t, err)
	assert.NotNil(t, formatted)
	assert.Equal(t, "coins", string(formatted.Execer))
	assert.NotEqual(t, int64(0), formatted.Nonce)
}

func TestSendAndCheckTx(t *testing.T) {
	cfg := testnode.GetDefaultConfig()
	mocker := testnode.NewWithConfig(cfg, nil)
	defer mocker.Close()
	mocker.Listen()
	err := mocker.SendHot()
	assert.Nil(t, err)

	// Create a simple transfer transaction
	txHex := "0a05636f696e73120e18010a0a1080c2d72f1a036f746520a08d0630f1cdebc8f7efa5e9283a22313271796f6361794e46374c7636433971573461767873324537553431664b536676"
	txBytes, _ := hex.DecodeString(txHex)
	var tx types.Transaction
	err = types.Decode(txBytes, &tx)
	assert.Nil(t, err)

	// Verify we can check this transaction
	api := mocker.GetAPI()
	_, err = api.GetProperFee(&types.ReqProperFee{TxCount: 1, TxSize: int32(types.Size(&tx))})
	assert.Nil(t, err)
}
