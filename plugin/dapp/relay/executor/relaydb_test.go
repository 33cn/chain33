package executor

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/db/mocks"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type suiteRelayLog struct {
	// Include our basic suite logic.
	suite.Suite
	db  *mocks.KV
	log *relayLog
}

func TestRunSuiteRelayLog(t *testing.T) {
	log := new(suiteRelayLog)
	suite.Run(t, log)
}

func (s *suiteRelayLog) SetupSuite() {
	order := &ty.RelayOrder{
		Id:         "123456",
		CoinTxHash: "aabbccddee",
	}
	s.db = new(mocks.KV)

	s.log = newRelayLog(order)
}

func (s *suiteRelayLog) TestSave() {
	kvSet := []*types.KeyValue{}
	value := types.Encode(&s.log.RelayOrder)
	keyId := []byte(s.log.Id)
	keyCoinTxHash := []byte(calcCoinHash(s.log.CoinTxHash))
	kvSet = append(kvSet, &types.KeyValue{keyId, value})
	kvSet = append(kvSet, &types.KeyValue{keyCoinTxHash, value})

	for i := 0; i < len(kvSet); i++ {
		s.db.On("Set", kvSet[i].GetKey(), kvSet[i].Value).Return(nil).Once()
	}

	rst := s.log.save(s.db)
	s.Equal(kvSet, rst)
}

func (s *suiteRelayLog) TestGetKVSet() {
	kvSet := []*types.KeyValue{}
	value := types.Encode(&s.log.RelayOrder)
	keyId := []byte(s.log.Id)
	keyCoinTxHash := []byte(calcCoinHash(s.log.CoinTxHash))
	kvSet = append(kvSet, &types.KeyValue{keyId, value})
	kvSet = append(kvSet, &types.KeyValue{keyCoinTxHash, value})

	rst := s.log.getKVSet()
	s.Assert().Equal(kvSet, rst)
}

func (s *suiteRelayLog) TestReceiptLog() {
	val := s.log.receiptLog(1)
	var log ty.ReceiptRelayLog
	types.Decode(val.Log, &log)
	s.Equal(s.log.Id, log.OrderId)
}

//////////////////////////////////////////////////
var (
	addrFrom = "14KEKbYtKKQm4wMthSK9J4La4nAiidGozt"
	addrTo   = "1Mcx9PczwPQ79tDnYzw62SEQifPwXH84yN"
	addrBtc  = "1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT"
)

//fromaddr 14KEKbYtKKQm4wMthSK9J4La4nAiidGozt
var privFrom = getprivkey("CC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")

//to 1Mcx9PczwPQ79tDnYzw62SEQifPwXH84yN
var privTo = getprivkey("BC38546E9E659D15E6B4893F0AB32A06D103931A8230B0BDE71459D2B27D6944")

func getprivkey(key string) crypto.PrivKey {
	cr, err := crypto.New(types.GetSignName("", types.SECP256K1))
	if err != nil {
		panic(err)
	}
	bkey, err := common.FromHex(key)
	if err != nil {
		panic(err)
	}
	priv, err := cr.PrivKeyFromBytes(bkey)
	if err != nil {
		panic(err)
	}
	return priv
}

type suiteRelayDB struct {
	// Include our basic suite logic.
	suite.Suite
	//db        *mocks.KV
	kvdb      *mocks.KVDB
	relay     *relay
	addrRelay string
	orderId   string
	relayDb   *relayDB
}

func (s *suiteRelayDB) accountSetup() {
	acc := s.relay.GetCoinsAccount()

	account := &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrFrom,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrFrom)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err := acc.TransferToExec(addrFrom, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)
	account = &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrTo,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrTo)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err = acc.TransferToExec(addrTo, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)
}

func (s *suiteRelayDB) SetupSuite() {
	//s.db = new(mocks.KV)
	s.kvdb = new(mocks.KVDB)
	accDb, _ := db.NewGoMemDB("relayTestDb", "test", 128)
	relay := &relay{}
	relay.SetStateDB(accDb)
	relay.SetLocalDB(s.kvdb)
	relay.SetEnv(10, 100, 1)
	relay.SetIsFree(false)
	relay.SetApi(nil)
	relay.SetChild(relay)
	s.relay = relay

	s.accountSetup()
}

func (s *suiteRelayDB) TestRelayCreate_1() {
	order := &ty.RelayCreate{
		Operation: ty.RelayOrderBuy,
		Coin:      "BTC",
		Amount:    10 * 1e8,
		Addr:      addrBtc,
		BtyAmount: 200 * 1e8,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Nonce = 1 //for different order id
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(10, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.create(order)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Frozen)
	s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(10), log.CoinHeight)
	s.orderId = log.OrderId
}

// the test suite function name need sequence so here aUnlock, bCancel
// unlock error
func (s *suiteRelayDB) TestRevokeCreate_1aUnlock() {
	order := &ty.RelayRevoke{
		OrderId: s.orderId,
		Target:  ty.RelayRevokeCreate,
		Action:  ty.RelayUnlock,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(11, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	_, err := s.relayDb.relayRevoke(order)
	s.Equal(ty.ErrRelayOrderParamErr, err)

}

func (s *suiteRelayDB) TestRevokeCreate_1bCancel() {
	order := &ty.RelayRevoke{
		OrderId: s.orderId,
		Target:  ty.RelayRevokeCreate,
		Action:  ty.RelayCancel,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(11, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	receipt, err := s.relayDb.relayRevoke(order)
	s.Nil(err)
	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal(ty.RelayOrderStatus_canceled.String(), log.CurStatus)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)
	s.Zero(account.Frozen)
}

func TestRunSuiteRelayDB(t *testing.T) {
	log := new(suiteRelayDB)
	suite.Run(t, log)
}

/////////////////////////////////////

type suiteAccept struct {
	// Include our basic suite logic.
	suite.Suite
	//db        *mocks.KV
	kvdb      *mocks.KVDB
	relay     *relay
	addrRelay string
	orderId   string
	relayDb   *relayDB
}

func (s *suiteAccept) setupAccount() {
	acc := s.relay.GetCoinsAccount()

	account := &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrFrom,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrFrom)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err := acc.TransferToExec(addrFrom, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)

	account = &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrTo,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrTo)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err = acc.TransferToExec(addrTo, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)

}

func (s *suiteAccept) setupRelayCreate() {
	order := &ty.RelayCreate{
		Operation: ty.RelayOrderBuy,
		Coin:      "BTC",
		Amount:    10 * 1e8,
		Addr:      addrBtc,
		BtyAmount: 200 * 1e8,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Nonce = 2 //for different order id
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(10, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.create(order)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Frozen)
	s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(10), log.CoinHeight)

	s.orderId = log.OrderId
}

func (s *suiteAccept) SetupSuite() {
	//s.db = new(mocks.KV)
	s.kvdb = new(mocks.KVDB)
	accDb, _ := db.NewGoMemDB("relayTestAccept", "test", 128)
	relay := &relay{}
	relay.SetStateDB(accDb)
	relay.SetLocalDB(s.kvdb)
	relay.SetEnv(10, 100, 1)
	relay.SetIsFree(false)
	relay.SetApi(nil)
	relay.SetChild(relay)
	s.relay = relay

	s.setupAccount()
	s.setupRelayCreate()
}

func (s *suiteAccept) TestRelayAccept() {

	order := &ty.RelayAccept{
		OrderId:  s.orderId,
		CoinAddr: "BTC",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(10, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(20)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.accept(order)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(20), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_locking.String(), log.CurStatus)

}

func (s *suiteAccept) TestRevokeAccept_1() {
	order := &ty.RelayRevoke{
		OrderId: s.orderId,
		Target:  ty.RelayRevokeAccept,
		Action:  ty.RelayUnlock,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(11, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(22)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	_, err := s.relayDb.relayRevoke(order)
	s.Equal(ty.ErrRelayBtcTxTimeErr, err)

}

func (s *suiteAccept) TestRevokeAccept_2() {
	order := &ty.RelayRevoke{
		OrderId: s.orderId,
		Target:  ty.RelayRevokeAccept,
		Action:  ty.RelayUnlock,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(11, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(20 + lockBtcHeight)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	_, err := s.relayDb.relayRevoke(order)
	s.Equal(ty.ErrRelayReturnAddr, err)

}

func (s *suiteAccept) TestRevokeAccept_3() {
	order := &ty.RelayRevoke{
		OrderId: s.orderId,
		Target:  ty.RelayRevokeAccept,
		Action:  ty.RelayUnlock,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(11, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(20 + lockBtcHeight)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	receipt, err := s.relayDb.relayRevoke(order)
	s.Nil(err)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal(uint64(20), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_pending.String(), log.CurStatus)
}

func TestRunSuiteAccept(t *testing.T) {
	log := new(suiteAccept)
	suite.Run(t, log)
}

/////////////////////////////////////////////

type suiteConfirm struct {
	// Include our basic suite logic.
	suite.Suite
	//db        *mocks.KV
	kvdb      *mocks.KVDB
	relay     *relay
	addrRelay string
	orderId   string
	relayDb   *relayDB
}

func (s *suiteConfirm) setupAccount() {
	acc := s.relay.GetCoinsAccount()

	account := &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrFrom,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrFrom)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err := acc.TransferToExec(addrFrom, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)

	account = &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrTo,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrTo)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err = acc.TransferToExec(addrTo, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)

}

func (s *suiteConfirm) setupRelayCreate() {
	order := &ty.RelayCreate{
		Operation: ty.RelayOrderBuy,
		Coin:      "BTC",
		Amount:    10 * 1e8,
		Addr:      addrBtc,
		BtyAmount: 200 * 1e8,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Nonce = 3 //for different order id
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(10, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.create(order)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Frozen)
	s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(10), log.CoinHeight)

	s.orderId = log.OrderId
}

func (s *suiteConfirm) SetupSuite() {
	//s.db = new(mocks.KV)
	s.kvdb = new(mocks.KVDB)
	accDb, _ := db.NewGoMemDB("relayTestAccept", "test", 128)
	relay := &relay{}
	relay.SetStateDB(accDb)
	relay.SetLocalDB(s.kvdb)
	relay.SetEnv(10, 100, 1)
	relay.SetIsFree(false)
	relay.SetApi(nil)
	relay.SetChild(relay)
	s.relay = relay

	s.setupAccount()
	s.setupRelayCreate()
	s.setupAccept()
}

func (s *suiteConfirm) setupAccept() {

	order := &ty.RelayAccept{
		OrderId:  s.orderId,
		CoinAddr: "BTC",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(10, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(20)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.accept(order)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(20), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_locking.String(), log.CurStatus)

}

func (s *suiteConfirm) TestConfirm_1() {

	order := &ty.RelayConfirmTx{
		OrderId: s.orderId,
		TxHash:  "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(30, 3000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	_, err := s.relayDb.confirmTx(order)
	s.Equal(ty.ErrRelayReturnAddr, err)

}

func (s *suiteConfirm) TestConfirm_2() {

	order := &ty.RelayConfirmTx{
		OrderId: s.orderId,
		TxHash:  "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(30, 3000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(30)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.confirmTx(order)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(30), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_confirming.String(), log.CurStatus)

}

func (s *suiteConfirm) TestRevokeConfirm_1() {
	order := &ty.RelayRevoke{
		OrderId: s.orderId,
		Target:  ty.RelayRevokeCreate,
		Action:  ty.RelayUnlock,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(40, 4000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(30 + lockBtcHeight)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	_, err := s.relayDb.relayRevoke(order)
	s.Equal(ty.ErrRelayBtcTxTimeErr, err)

}

func (s *suiteConfirm) TestRevokeConfirm_2() {
	order := &ty.RelayRevoke{
		OrderId: s.orderId,
		Target:  ty.RelayRevokeCreate,
		Action:  ty.RelayUnlock,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(40, 4000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(30 + 4*lockBtcHeight)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	receipt, err := s.relayDb.relayRevoke(order)
	s.Nil(err)
	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal(uint64(30), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_pending.String(), log.CurStatus)

}

func TestRunSuiteConfirm(t *testing.T) {
	log := new(suiteConfirm)
	suite.Run(t, log)
}

/////////////////////////////////////////////

type suiteVerify struct {
	// Include our basic suite logic.
	suite.Suite
	//db        *mocks.KV
	kvdb      *mocks.KVDB
	relay     *relay
	addrRelay string
	orderId   string
	relayDb   *relayDB
}

func (s *suiteVerify) setupAccount() {
	acc := s.relay.GetCoinsAccount()

	account := &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrFrom,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrFrom)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err := acc.TransferToExec(addrFrom, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)

	account = &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrTo,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrTo)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err = acc.TransferToExec(addrTo, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)

}

func (s *suiteVerify) setupRelayCreate() {
	order := &ty.RelayCreate{
		Operation: ty.RelayOrderBuy,
		Coin:      "BTC",
		Amount:    0.299 * 1e8,
		Addr:      addrBtc,
		BtyAmount: 200 * 1e8,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Nonce = 4 //for different order id
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(10, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.create(order)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Frozen)
	s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(10), log.CoinHeight)

	s.orderId = log.OrderId
}

func (s *suiteVerify) setupAccept() {

	order := &ty.RelayAccept{
		OrderId:  s.orderId,
		CoinAddr: "BTC",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(20, 2000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(20)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.accept(order)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(20), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_locking.String(), log.CurStatus)

}

func (s *suiteVerify) setupConfirm() {

	order := &ty.RelayConfirmTx{
		OrderId: s.orderId,
		TxHash:  "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(30, 3000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(30)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.confirmTx(order)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(30), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_confirming.String(), log.CurStatus)

}

func (s *suiteVerify) SetupSuite() {
	//s.db = new(mocks.KV)
	s.kvdb = new(mocks.KVDB)
	accDb, _ := db.NewGoMemDB("relayTestAccept", "test", 128)
	relay := &relay{}
	relay.SetStateDB(accDb)
	relay.SetLocalDB(s.kvdb)
	relay.SetEnv(10, 100, 1)
	relay.SetIsFree(false)
	relay.SetApi(nil)
	relay.SetChild(relay)
	s.relay = relay

	s.setupAccount()
	s.setupRelayCreate()
	s.setupAccept()
	s.setupConfirm()
}

func (s *suiteVerify) TestVerify() {
	vout := &ty.Vout{
		Address: "1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT",
		Value:   29900000,
	}
	transaction := &ty.BtcTransaction{
		Vout:        []*ty.Vout{vout},
		Time:        2500,
		BlockHeight: 1000,
		Hash:        "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}
	str_merkleproof := []string{"e9a66845e05d5abc0ad04ec80f774a7e585c6e8db975962d069a522137b80c1d",
		"ccdafb73d8dcd0173d5d5c3c9a0770d0b3953db889dab99ef05b1907518cb815"}

	proofs := make([][]byte, len(str_merkleproof))
	for i, kk := range str_merkleproof {
		proofs[i], _ = btcHashStrRevers(kk)
	}

	spv := &ty.BtcSpv{
		BranchProof: proofs,
		TxIndex:     2,
		BlockHash:   "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506",
		Height:      100000,
		Hash:        "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}

	heightBytes := types.Encode(&types.Int64{int64(1006)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	var head = &ty.BtcHeader{
		Version:    1,
		MerkleRoot: "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
	}
	headEnc := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(headEnc, nil).Once()

	order := &ty.RelayVerify{
		OrderId: s.orderId,
		Tx:      transaction,
		Spv:     spv,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(40, 4000, 1)
	s.relayDb = newRelayDB(s.relay, tx)

	receipt, err := s.relayDb.verifyTx(order)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Balance)
	s.Zero(account.Frozen)
	account = acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(400*1e8), account.Balance)
	s.Zero(account.Frozen)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	//s.Equal("200.0000",log.TxAmount)
	//s.Equal(uint64(30),log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_finished.String(), log.CurStatus)

}

func TestRunSuiteVerify(t *testing.T) {
	log := new(suiteVerify)
	suite.Run(t, log)
}

/////////////////////////////////////////////

type suiteVerifyCli struct {
	// Include our basic suite logic.
	suite.Suite
	//db        *mocks.KV
	kvdb      *mocks.KVDB
	relay     *relay
	addrRelay string
	orderId   string
	relayDb   *relayDB
}

func (s *suiteVerifyCli) setupAccount() {
	acc := s.relay.GetCoinsAccount()

	account := &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrFrom,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrFrom)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err := acc.TransferToExec(addrFrom, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)

	account = &types.Account{
		Balance: 1000 * 1e8,
		Addr:    addrTo,
	}
	acc.SaveAccount(account)
	account = acc.LoadAccount(addrTo)
	s.Equal(int64(1000*1e8), account.Balance)
	s.addrRelay = address.ExecAddress("relay")
	_, err = acc.TransferToExec(addrTo, s.addrRelay, 200*1e8)
	s.Nil(err)
	account = acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(200*1e8), account.Balance)
}

func (s *suiteVerifyCli) setupRelayCreate() {
	order := &ty.RelayCreate{
		Operation: ty.RelayOrderBuy,
		Coin:      "BTC",
		Amount:    0.299 * 1e8,
		Addr:      addrBtc,
		BtyAmount: 200 * 1e8,
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Nonce = 5 //for different order id
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(10, 1000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.create(order)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(200*1e8), account.Frozen)
	s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(10), log.CoinHeight)

	s.orderId = log.OrderId
}

func (s *suiteVerifyCli) setupAccept() {

	order := &ty.RelayAccept{
		OrderId:  s.orderId,
		CoinAddr: "BTC",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(20, 2000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(20)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.accept(order)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(20), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_locking.String(), log.CurStatus)

}

func (s *suiteVerifyCli) setupConfirm() {

	order := &ty.RelayConfirmTx{
		OrderId: s.orderId,
		TxHash:  "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(30, 3000, 1)
	s.relayDb = newRelayDB(s.relay, tx)
	heightBytes := types.Encode(&types.Int64{int64(30)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relayDb.confirmTx(order)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(30), log.CoinHeight)
	s.Equal(ty.RelayOrderStatus_confirming.String(), log.CurStatus)

}

func (s *suiteVerifyCli) SetupSuite() {
	//s.db = new(mocks.KV)
	s.kvdb = new(mocks.KVDB)
	accDb, _ := db.NewGoMemDB("relayTestAccept", "test", 128)
	relay := &relay{}
	relay.SetStateDB(accDb)
	relay.SetLocalDB(s.kvdb)
	relay.SetEnv(10, 100, 1)
	relay.SetIsFree(false)
	relay.SetApi(nil)
	relay.SetChild(relay)
	s.relay = relay

	s.setupAccount()
	s.setupRelayCreate()
	s.setupAccept()
	s.setupConfirm()
}

func (s *suiteVerifyCli) TestVerify() {
	var head = &ty.BtcHeader{
		Version:    1,
		MerkleRoot: "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
	}
	headEnc := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(headEnc, nil).Once()

	order := &ty.RelayVerifyCli{
		OrderId:    s.orderId,
		RawTx:      "0100000001c33ebff2a709f13d9f9a7569ab16a32786af7d7e2de09265e41c61d078294ecf010000008a4730440220032d30df5ee6f57fa46cddb5eb8d0d9fe8de6b342d27942ae90a3231e0ba333e02203deee8060fdc70230a7f5b4ad7d7bc3e628cbe219a886b84269eaeb81e26b4fe014104ae31c31bf91278d99b8377a35bbce5b27d9fff15456839e919453fc7b3f721f0ba403ff96c9deeb680e5fd341c0fc3a7b90da4631ee39560639db462e9cb850fffffffff0240420f00000000001976a914b0dcbf97eabf4404e31d952477ce822dadbe7e1088acc060d211000000001976a9146b1281eec25ab4e1e0793ff4e08ab1abb3409cd988ac00000000",
		TxIndex:    2,
		MerkBranch: "e9a66845e05d5abc0ad04ec80f774a7e585c6e8db975962d069a522137b80c1d-ccdafb73d8dcd0173d5d5c3c9a0770d0b3953db889dab99ef05b1907518cb815",
		BlockHash:  "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506",
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(40, 4000, 1)
	s.relayDb = newRelayDB(s.relay, tx)

	receipt, err := s.relayDb.verifyCmdTx(order)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Balance)
	s.Zero(account.Frozen)
	account = acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(400*1e8), account.Balance)
	s.Zero(account.Frozen)

	var log ty.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)

	s.Equal(ty.RelayOrderStatus_finished.String(), log.CurStatus)

}

func TestRunSuiteVerifyCli(t *testing.T) {
	log := new(suiteVerifyCli)
	suite.Run(t, log)
}

/////////////////////////////////////////////

type suiteSaveBtcHeader struct {
	// Include our basic suite logic.
	suite.Suite
	db      *mocks.KV
	kvdb    *mocks.KVDB
	relay   *relay
	relayDb *relayDB
}

func (s *suiteSaveBtcHeader) SetupSuite() {
	s.db = new(mocks.KV)
	s.kvdb = new(mocks.KVDB)
	relay := &relay{}
	relay.SetStateDB(s.db)
	relay.SetLocalDB(s.kvdb)
	relay.SetEnv(10, 100, 1)
	relay.SetIsFree(false)
	relay.SetApi(nil)
	relay.SetChild(relay)
	s.relay = relay

	tx := &types.Transaction{}
	tx.To = "addr"
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(40, 4000, 1)
	s.relayDb = newRelayDB(s.relay, tx)

}

func (s *suiteSaveBtcHeader) TestSaveBtcHeader_1() {
	head0 := &ty.BtcHeader{
		Hash:          "5e7d9c599cd040ec2ba53f4dee28028710be8c135e779f65c56feadaae34c3f2",
		Confirmations: 92,
		Height:        10,
		Version:       536870912,
		MerkleRoot:    "ab91cd4160e1379c337eee6b7a4bdbb7399d70268d86045aba150743c00c90b6",
		Time:          1530862108,
		Nonce:         0,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "604efe53975ab06cad8748fd703ad5bc960e8b752b2aae98f0f871a4a05abfc7",
	}
	head1 := &ty.BtcHeader{
		Hash:          "7b7a4a9b49db5a1162be515d380cd186e98c2bf0bb90f1145485d7c43343fc7c",
		Confirmations: 91,
		Height:        11,
		Version:       536870912,
		MerkleRoot:    "cfa9b66696aea63b7266ffaa1cb4b96c8dd6959eaabf2eb14173f4adaa551f6f",
		Time:          1530862108,
		Nonce:         1,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "5e7d9c599cd040ec2ba53f4dee28028710be8c135e779f65c56feadaae34c3f2",
	}

	head2 := &ty.BtcHeader{
		Hash:          "57bd2805725dd2d102708af4c8f6eb67cd0b3de6dd531f59fbc7d441a0388b6e",
		Confirmations: 90,
		Height:        12,
		Version:       536870912,
		MerkleRoot:    "f549e7024c07b741d34633654847be8778d2ccc9b1b17571914050060cc785f6",
		Time:          1530862108,
		Nonce:         4,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "7b7a4a9b49db5a1162be515d380cd186e98c2bf0bb90f1145485d7c43343fc7c",
	}

	headers := &ty.BtcHeaders{}
	headers.BtcHeader = append(headers.BtcHeader, head0)
	headers.BtcHeader = append(headers.BtcHeader, head1)
	headers.BtcHeader = append(headers.BtcHeader, head2)

	s.db.On("Get", mock.Anything).Return(nil, types.ErrNotFound).Once()
	s.db.On("Set", mock.Anything, mock.Anything).Return(nil).Once()
	receipt, err := s.relayDb.saveBtcHeader(headers, s.kvdb)
	s.Nil(err)
	var log ty.ReceiptRelayRcvBTCHeaders
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)

	s.Zero(log.LastHeight)
	s.Zero(log.LastBaseHeight)
	s.Equal(head2.Height, log.NewHeight)
	s.Zero(log.NewBaseHeight)
	s.Equal(headers.BtcHeader, log.Headers)

}

//not continuous
func (s *suiteSaveBtcHeader) TestSaveBtcHeader_2() {
	head3 := &ty.BtcHeader{
		Hash:          "16ad6d588aeca12bf3e7fd6b2263992b4442c9692f4e134ba7cf0d791746328a",
		Confirmations: 89,
		Height:        13,
		Version:       536870912,
		MerkleRoot:    "0921a87a7a196a54d3cf3f89b2dc218d6451d3fd1ce719db4d6c3f74746cb63c",
		Time:          1530862108,
		Nonce:         0,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "67bd2805725dd2d102708af4c8f6eb67cd0b3de6dd531f59fbc7d441a0388b6e",
	}
	head4 := &ty.BtcHeader{
		Hash:          "3bc0ee712c84c589c693b09aa3685f266c2c295a6d031952ecc863a2a1eefe45",
		Confirmations: 89,
		Height:        14,
		Version:       536870912,
		MerkleRoot:    "05a9866eda375bc91a404aa10f3fface31d3494716ead686fe72ebc46793b0ba",
		Time:          1530862109,
		Nonce:         2,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "16ad6d588aeca12bf3e7fd6b2263992b4442c9692f4e134ba7cf0d791746328a",
	}

	headers := &ty.BtcHeaders{}
	headers.BtcHeader = append(headers.BtcHeader, head3)
	headers.BtcHeader = append(headers.BtcHeader, head4)
	s.db.On("Get", mock.Anything).Return(nil, types.ErrNotFound).Once()
	_, err := s.relayDb.saveBtcHeader(headers, s.kvdb)
	s.Equal(ty.ErrRelayBtcHeadHashErr, err)

}

//not continuous than previous
func (s *suiteSaveBtcHeader) TestSaveBtcHeader_3() {
	head3 := &ty.BtcHeader{
		Hash:          "16ad6d588aeca12bf3e7fd6b2263992b4442c9692f4e134ba7cf0d791746328a",
		Confirmations: 89,
		Height:        13,
		Version:       536870912,
		MerkleRoot:    "0921a87a7a196a54d3cf3f89b2dc218d6451d3fd1ce719db4d6c3f74746cb63c",
		Time:          1530862108,
		Nonce:         0,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "67bd2805725dd2d102708af4c8f6eb67cd0b3de6dd531f59fbc7d441a0388b6e",
	}
	head4 := &ty.BtcHeader{
		Hash:          "3bc0ee712c84c589c693b09aa3685f266c2c295a6d031952ecc863a2a1eefe45",
		Confirmations: 89,
		Height:        14,
		Version:       536870912,
		MerkleRoot:    "05a9866eda375bc91a404aa10f3fface31d3494716ead686fe72ebc46793b0ba",
		Time:          1530862109,
		Nonce:         2,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "16ad6d588aeca12bf3e7fd6b2263992b4442c9692f4e134ba7cf0d791746328a",
	}

	headers := &ty.BtcHeaders{}
	headers.BtcHeader = append(headers.BtcHeader, head3)

	lastHead := &ty.RelayLastRcvBtcHeader{
		Header:     head4,
		BaseHeight: 10,
	}

	head4Encode := types.Encode(lastHead)
	s.db.On("Get", mock.Anything).Return(head4Encode, nil).Once()

	_, err := s.relayDb.saveBtcHeader(headers, s.kvdb)
	s.Equal(ty.ErrRelayBtcHeadSequenceErr, err)

}

//reset
func (s *suiteSaveBtcHeader) TestSaveBtcHeader_4() {

	head4 := &ty.BtcHeader{
		Hash:          "3bc0ee712c84c589c693b09aa3685f266c2c295a6d031952ecc863a2a1eefe45",
		Confirmations: 89,
		Height:        14,
		Version:       536870912,
		MerkleRoot:    "05a9866eda375bc91a404aa10f3fface31d3494716ead686fe72ebc46793b0ba",
		Time:          1530862109,
		Nonce:         2,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "16ad6d588aeca12bf3e7fd6b2263992b4442c9692f4e134ba7cf0d791746328a",
		IsReset:       true,
	}

	head5 := &ty.BtcHeader{
		Hash:          "439e515ee8307fccc104395a46626bd43f6042482d56f4abcfaeee3c0a1b1ece",
		Confirmations: 89,
		Height:        15,
		Version:       536870912,
		MerkleRoot:    "07a3c184d6f9813f468f0890f1da4449844e684a819c91cb58d2d5a74eeaf5a9",
		Time:          1530862109,
		Nonce:         2,
		Bits:          545259519,
		Difficulty:    0,
		PreviousHash:  "3bc0ee712c84c589c693b09aa3685f266c2c295a6d031952ecc863a2a1eefe45",
	}

	headers := &ty.BtcHeaders{}
	headers.BtcHeader = append(headers.BtcHeader, head4)
	headers.BtcHeader = append(headers.BtcHeader, head5)

	s.db.On("Get", mock.Anything).Return(nil, types.ErrNotFound).Once()
	s.db.On("Set", mock.Anything, mock.Anything).Return(nil).Once()
	receipt, err := s.relayDb.saveBtcHeader(headers, s.kvdb)
	s.Nil(err)
	var log ty.ReceiptRelayRcvBTCHeaders
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)

	s.Zero(log.LastHeight)
	s.Zero(log.LastBaseHeight)
	s.Equal(head5.Height, log.NewHeight)
	s.Equal(head4.Height, log.NewBaseHeight)
	s.Equal(headers.BtcHeader, log.Headers)

}

func TestRunSuiteSaveBtcHeader(t *testing.T) {
	log := new(suiteSaveBtcHeader)
	suite.Run(t, log)
}
