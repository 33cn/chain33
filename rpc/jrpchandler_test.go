package rpc

import (
	"testing"

	"github.com/go-siris/siris/core/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gitlab.33.cn/chain33/chain33/client/mocks"
	"gitlab.33.cn/chain33/chain33/types"
)

func TestDecodeUserWrite(t *testing.T) {
	payload := []byte("#md#hello#world")
	data := decodeUserWrite(payload)
	assert.Equal(t, data, &userWrite{Topic: "md", Content: "hello#world"})

	payload = []byte("hello#world")
	data = decodeUserWrite(payload)
	assert.Equal(t, data, &userWrite{Topic: "", Content: "hello#world"})

	payload = []byte("123#hello#suyanlong")
	data = decodeUserWrite(payload)
	assert.NotEqual(t, data, &userWrite{Topic: "123", Content: "hello#world"})
}

func TestDecodeTx(t *testing.T) {
	tx := types.Transaction{
		Execer:  []byte("coin"),
		Payload: []byte("342412abcd"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	data, err := DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)

	tx.Execer = []byte("coins")
	data, err = DecodeTx(&tx)
	assert.NotNil(t, err)
	assert.Nil(t, data)

	tx = types.Transaction{
		Execer:  []byte("hashlock"),
		Payload: []byte("34"),
		Nonce:   8978167239,
		To:      "1asd234dsf43fds",
	}

	t.Log(string(tx.Execer))
	data, err = DecodeTx(&tx)
	assert.NotNil(t, data)
	assert.Nil(t, err)
}

func TestDecodeLog(t *testing.T) {
	var data = &ReceiptData{
		Ty: 5,
	}
	result, err := DecodeLog(data)
	assert.Nil(t, err)
	assert.NotNil(t, result)
}

func newTestChain33(api *mocks.QueueProtocolAPI) *Chain33 {
	return &Chain33{
		cli: channelClient{
			QueueProtocolAPI: api,
		},
	}
}

func TestChain33_CreateRawTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	// var result interface{}
	// api.On("CreateRawTransaction", nil, &result).Return()
	testChain33 := newTestChain33(api)
	var testResult interface{}
	err := testChain33.CreateRawTransaction(nil, &testResult)
	assert.Nil(t, testResult)
	assert.NotNil(t, err)

	tx := &types.CreateTx{
		To:          "qew",
		Amount:      10,
		Fee:         1,
		Note:        "12312",
		IsWithdraw:  false,
		IsToken:     true,
		TokenSymbol: "CNY",
		ExecName:    "token",
	}

	err = testChain33.CreateRawTransaction(tx, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_SendRawTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	// var result interface{}
	api.On("SendTx", mock.Anything).Return()

	testChain33 := newTestChain33(api)
	var testResult interface{}
	signedTx := SignedTx{
		Unsign: "123",
		Sign:   "123",
		Pubkey: "123",
		Ty:     1,
	}
	err := testChain33.SendRawTransaction(signedTx, &testResult)
	t.Log(err)
	assert.Nil(t, testResult)
	assert.NotNil(t, err)
	// api.Called(1)
	// mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SendTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("SendTx", &types.Transaction{}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := RawParm{
		Data: "",
	}
	err := testChain33.SendTransaction(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetHexTxByHash(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("QueryTx", &types.ReqHash{}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := QueryParm{
		Hash: "",
	}
	err := testChain33.GetHexTxByHash(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_QueryTransaction(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("QueryTx", &types.ReqHash{}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := QueryParm{
		Hash: "",
	}
	err := testChain33.QueryTransaction(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetBlocks(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("GetBlocks", &types.ReqBlocks{Pid: []string{""}}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := BlockParam{}
	err := testChain33.GetBlocks(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetLastHeader(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	api.On("GetBlocks", &types.ReqBlocks{Pid: []string{""}}).Return(nil, errors.New("error value"))
	testChain33 := newTestChain33(api)
	var testResult interface{}
	data := BlockParam{}
	err := testChain33.GetBlocks(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetTxByAddr(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("GetTransactionByAddr", &types.ReqAddr{}).Return(nil, errors.New("error value"))
	var testResult interface{}
	data := types.ReqAddr{}
	err := testChain33.GetTxByAddr(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetTxByHashes(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	var parm types.ReqHashes
	parm.Hashes = make([][]byte, 0)
	api.On("GetTransactionByHash", &parm).Return(nil, errors.New("error value"))
	var testResult interface{}
	data := ReqHashes{}
	err := testChain33.GetTxByHashes(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetMempool(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("GetMempool").Return(nil, errors.New("error value"))
	var testResult interface{}
	data := &types.ReqNil{}
	err := testChain33.GetMempool(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetAccounts(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("WalletGetAccountList").Return(nil, errors.New("error value"))
	var testResult interface{}
	data := &types.ReqNil{}
	err := testChain33.GetAccounts(data, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_NewAccount(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("NewAccount", &types.ReqNewAccount{}).Return(nil, errors.New("error value"))

	var testResult interface{}
	err := testChain33.NewAccount(types.ReqNewAccount{}, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_WalletTxList(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletTransactionList{FromTx: []byte("")}
	api.On("WalletTransactionList", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := ReqWalletTransactionList{}
	err := testChain33.WalletTxList(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_ImportPrivkey(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletImportPrivKey{}
	api.On("WalletImportprivkey", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletImportPrivKey{}
	err := testChain33.ImportPrivkey(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SendToAddress(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSendToAddress{}
	api.On("WalletSendToAddress", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSendToAddress{}
	err := testChain33.SendToAddress(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SetTxFee(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSetFee{}
	api.On("WalletSetFee", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSetFee{}
	err := testChain33.SetTxFee(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SetLabl(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSetLabel{}
	api.On("WalletSetLabel", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSetLabel{}
	err := testChain33.SetLabl(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_MergeBalance(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletMergeBalance{}
	api.On("WalletMergeBalance", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletMergeBalance{}
	err := testChain33.MergeBalance(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SetPasswd(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqWalletSetPasswd{}
	api.On("WalletSetPasswd", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqWalletSetPasswd{}
	err := testChain33.SetPasswd(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_Lock(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	// expected := types.ReqNil{}
	api.On("WalletLock").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.Lock(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_UnLock(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.WalletUnLock{}
	api.On("WalletUnLock", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.WalletUnLock{}
	err := testChain33.UnLock(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetPeerInfo(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	api.On("PeerInfo").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.GetPeerInfo(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetHeaders(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqBlocks{}
	api.On("GetHeaders", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqBlocks{}
	err := testChain33.GetHeaders(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetLastMemPool(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	// expected := &types.ReqBlocks{}
	api.On("GetLastMempool").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.GetLastMemPool(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetBlockOverview(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqHash{}
	api.On("GetBlockOverview", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := QueryParm{}
	err := testChain33.GetBlockOverview(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetAddrOverview(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqAddr{}
	api.On("GetAddrOverview", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqAddr{}
	err := testChain33.GetAddrOverview(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	// mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetBlockHash(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.ReqInt{}
	api.On("GetBlockHash", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqInt{}
	err := testChain33.GetBlockHash(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GenSeed(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.GenSeedLang{}
	api.On("GenSeed", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.GenSeedLang{}
	err := testChain33.GenSeed(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_SaveSeed(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.SaveSeedByPw{}
	api.On("SaveSeed", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.SaveSeedByPw{}
	err := testChain33.SaveSeed(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetSeed(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	expected := &types.GetSeedByPw{}
	api.On("GetSeed", expected).Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.GetSeedByPw{}
	err := testChain33.GetSeed(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

func TestChain33_GetWalletStatus(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)

	// expected := &types.GetSeedByPw{}
	api.On("GetWalletStatus").Return(nil, errors.New("error value"))

	var testResult interface{}
	actual := types.ReqNil{}
	err := testChain33.GetWalletStatus(actual, &testResult)
	t.Log(err)
	assert.Equal(t, nil, testResult)
	assert.NotNil(t, err)

	mock.AssertExpectationsForObjects(t, api)
}

// func TestChain33_GetBalance(t *testing.T) {
// 	api := new(mocks.QueueProtocolAPI)
// 	testChain33 := newTestChain33(api)
//
// 	expected := &types.ReqBalance{}
// 	api.On("GetBalance",expected).Return(nil, errors.New("error value"))
//
// 	var testResult interface{}
// 	actual := types.ReqBalance{}
// 	err := testChain33.GetBalance(actual, &testResult)
// 	t.Log(err)
// 	assert.Equal(t, nil, testResult)
// 	assert.NotNil(t, err)
//
// 	mock.AssertExpectationsForObjects(t, api)
// }

// ----------------------------

func TestChain33_Version(t *testing.T) {
	api := new(mocks.QueueProtocolAPI)
	testChain33 := newTestChain33(api)
	var testResult interface{}
	in := &types.ReqNil{}
	err := testChain33.Version(in, &testResult)
	t.Log(err)
	assert.Equal(t, nil, err)
	assert.NotNil(t, testResult)
}

func TestChain33_CreateRawTokenPreCreateTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTokenPreCreateTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TokenPreCreateTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	err = client.CreateRawTokenPreCreateTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTokenRevokeTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTokenRevokeTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TokenRevokeTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	err = client.CreateRawTokenRevokeTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTokenFinishTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTokenFinishTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TokenFinishTx{
		OwnerAddr: "asdf134",
		Symbol:    "CNY",
		Fee:       123,
	}
	err = client.CreateRawTokenFinishTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeSellTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeSellTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeSellTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}

	err = client.CreateRawTradeSellTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeBuyTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeBuyTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeBuyTx{
		SellID:      "sadfghjkhgfdsa",
		BoardlotCnt: 100,
		Fee:         1,
	}

	err = client.CreateRawTradeBuyTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}

func TestChain33_CreateRawTradeRevokeTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeRevokeTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeRevokeTx{
		SellID: "sadfghjkhgfdsa",
		Fee:    1,
	}

	err = client.CreateRawTradeRevokeTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeBuyLimitTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeBuyLimitTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeBuyLimitTx{
		TokenSymbol:       "CNY",
		AmountPerBoardlot: 10,
		MinBoardlot:       1,
		PricePerBoardlot:  100,
		TotalBoardlot:     100,
		Fee:               1,
	}

	err = client.CreateRawTradeBuyLimitTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeSellMarketTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeSellMarketTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeSellMarketTx{
		BuyID:       "12asdfa",
		BoardlotCnt: 100,
		Fee:         1,
	}

	err = client.CreateRawTradeSellMarketTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)

}

func TestChain33_CreateRawTradeRevokeBuyTx(t *testing.T) {
	client := newTestChain33(nil)
	var testResult interface{}
	err := client.CreateRawTradeRevokeBuyTx(nil, &testResult)
	assert.NotNil(t, err)
	assert.Nil(t, testResult)

	token := &TradeRevokeBuyTx{
		BuyID: "12asdfa",
		Fee:   1,
	}

	err = client.CreateRawTradeRevokeBuyTx(token, &testResult)
	assert.NotNil(t, testResult)
	assert.Nil(t, err)
}
