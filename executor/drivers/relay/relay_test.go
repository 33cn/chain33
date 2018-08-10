package relay

import (
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/db"
	"gitlab.33.cn/chain33/chain33/common/db/mocks"
	"gitlab.33.cn/chain33/chain33/types"
)

type suiteRelay struct {
	// Include our basic suite logic.
	suite.Suite
	kvdb      *mocks.KVDB
	relay     *relay
	addrRelay string
	orderId   string
	//relayDb   *relayDB
}

func (s *suiteRelay) accountSetup() {
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

func (s *suiteRelay) SetupSuite() {
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

	s.Equal("relay", s.relay.GetName())

	s.accountSetup()
}

func (s *suiteRelay) testExecLocal(tx *types.Transaction, receipt *types.Receipt) {
	s.Equal(int32(types.ExecOk), receipt.Ty)
	rData := &types.ReceiptData{}
	rData.Ty = receipt.Ty
	rData.Logs = append(rData.Logs, receipt.Logs...)
	s.kvdb.On("Get", mock.Anything).Return([]byte{}, nil).Twice()

	set, err := s.relay.ExecLocal(tx, rData, 0)
	s.Nil(err)
	order, _ := s.relay.getSellOrderFromDb([]byte(s.orderId))
	var kv []*types.KeyValue
	kv = deleteCreateOrderKeyValue(kv, order, int32(order.PreStatus))
	kv = getCreateOrderKeyValue(kv, order, int32(order.Status))

	s.Subset(set.KV, kv)

}

func (s *suiteRelay) testExecDelLocal(tx *types.Transaction, receipt *types.Receipt) {
	s.Equal(int32(types.ExecOk), receipt.Ty)
	rData := &types.ReceiptData{}
	rData.Ty = receipt.Ty
	rData.Logs = append(rData.Logs, receipt.Logs...)
	s.kvdb.On("Get", mock.Anything).Return([]byte{}, nil).Twice()

	set, err := s.relay.ExecDelLocal(tx, rData, 0)
	s.Nil(err)
	order, _ := s.relay.getSellOrderFromDb([]byte(s.orderId))
	var kv []*types.KeyValue
	kv = deleteCreateOrderKeyValue(kv, order, int32(order.Status))
	kv = getCreateOrderKeyValue(kv, order, int32(order.PreStatus))

	s.Subset(set.KV, kv)
}

//create sell
func (s *suiteRelay) TestExec_1() {
	order := &types.RelayCreate{
		Operation: types.RelayOrderSell,
		Coin:      "BTC",
		Amount:    0.299 * 1e8,
		Addr:      addrBtc,
		BtyAmount: 200 * 1e8,
	}

	sell := &types.RelayAction{
		Ty:    types.RelayActionCreate,
		Value: &types.RelayAction_Create{order},
	}

	tx := &types.Transaction{}
	tx.Execer = types.ExecerRelay
	tx.To = address.ExecAddress(string(types.ExecerRelay))
	tx.Nonce = 1 //for different order id
	tx.Payload = types.Encode(sell)
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(10, 1000, 1)
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()

	receipt, err := s.relay.Exec(tx, 0)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8), account.Frozen)
	//s.Zero(account.Balance)

	var log types.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(10), log.CoinHeight)

	s.orderId = log.OrderId

	s.testExecLocal(tx, receipt)
	s.testExecDelLocal(tx, receipt)

}

//accept
func (s *suiteRelay) TestExec_2() {
	order := &types.RelayAccept{
		OrderId:  s.orderId,
		CoinAddr: addrBtc,
	}

	sell := &types.RelayAction{
		Ty:    types.RelayActionAccept,
		Value: &types.RelayAction_Accept{order},
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Payload = types.Encode(sell)
	tx.Sign(types.SECP256K1, privTo)

	s.relay.SetEnv(20, 2000, 1)
	heightBytes := types.Encode(&types.Int64{int64(20)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relay.Exec(tx, 0)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrTo, s.addrRelay)
	s.Equal(int64(200*1e8), account.Frozen)
	s.Zero(account.Balance)

	var log types.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(20), log.CoinHeight)
	s.Equal(types.RelayOrderStatus_locking.String(), log.CurStatus)

}

//confirm
func (s *suiteRelay) TestExec_3() {

	order := &types.RelayConfirmTx{
		OrderId: s.orderId,
		TxHash:  "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}
	sell := &types.RelayAction{
		Ty:    types.RelayActionConfirmTx,
		Value: &types.RelayAction_ConfirmTx{order},
	}

	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Payload = types.Encode(sell)
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(30, 3000, 1)
	heightBytes := types.Encode(&types.Int64{int64(30)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	receipt, err := s.relay.Exec(tx, 0)
	s.Nil(err)

	//acc := s.relay.GetCoinsAccount()
	//account := acc.LoadExecAccount(addrFrom, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Frozen)
	//s.Zero(account.Balance)

	var log types.ReceiptRelayLog
	types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	s.Equal("200.0000", log.TxAmount)
	s.Equal(uint64(30), log.CoinHeight)
	s.Equal(types.RelayOrderStatus_confirming.String(), log.CurStatus)

}

//verify

func (s *suiteRelay) TestExec_4() {
	vout := &types.Vout{
		Address: "1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT",
		Value:   29900000,
	}
	transaction := &types.BtcTransaction{
		Vout:        []*types.Vout{vout},
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

	spv := &types.BtcSpv{
		BranchProof: proofs,
		TxIndex:     2,
		BlockHash:   "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506",
		Height:      100000,
		Hash:        "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}

	heightBytes := types.Encode(&types.Int64{int64(1006)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	var head = &types.BtcHeader{
		Version:    1,
		MerkleRoot: "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
	}
	headEnc := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(headEnc, nil).Once()

	order := &types.RelayVerify{
		OrderId: s.orderId,
		Tx:      transaction,
		Spv:     spv,
	}
	sell := &types.RelayAction{
		Ty:    types.RelayActionVerifyTx,
		Value: &types.RelayAction_Verify{order},
	}
	tx := &types.Transaction{}
	tx.To = s.addrRelay
	tx.Payload = types.Encode(sell)
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(40, 4000, 1)

	_, err := s.relay.Exec(tx, 0)
	s.Nil(err)

	acc := s.relay.GetCoinsAccount()
	account := acc.LoadExecAccount(addrTo, s.addrRelay)
	//s.Equal(int64(200*1e8),account.Balance)
	s.Zero(account.Frozen)
	account = acc.LoadExecAccount(addrFrom, s.addrRelay)
	s.Equal(int64(400*1e8), account.Balance)
	s.Zero(account.Frozen)

	//	s.T().Log("exec4",len(receipt.Logs))
	//var log types.ReceiptRelayLog
	//types.Decode(receipt.Logs[len(receipt.Logs)-1].Log, &log)
	//s.Equal(types.RelayOrderStatus_finished.String(), log.CurStatus)

}

func (s *suiteRelay) TestExec_9_QryStatus1() {
	addrCoins := &types.ReqRelayAddrCoins{
		Status: types.RelayOrderStatus_finished,
		//Coins:[]string{"BTC"},
	}

	var OrderIds [][]byte
	OrderIds = append(OrderIds, []byte(s.orderId))
	s.kvdb.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(OrderIds, nil).Once()
	param := types.Encode(addrCoins)
	msg, err := s.relay.Query("GetRelayOrderByStatus", param)
	s.Nil(err)
	//s.T().Log(msg.String())
	s.Contains(msg.String(), "status:finished")
	//s.Equal(types.RelayOrderStatus_finished,)
}

func (s *suiteRelay) TestExec_9_QryStatus2() {
	addrCoins := &types.ReqRelayAddrCoins{
		Addr: addrFrom,
		//Status: types.RelayOrderStatus_finished,
		Coins: []string{"BTC"},
	}

	var OrderIds [][]byte
	OrderIds = append(OrderIds, []byte(s.orderId))
	s.kvdb.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(OrderIds, nil).Once()
	param := types.Encode(addrCoins)
	msg, err := s.relay.Query("GetSellRelayOrder", param)
	s.Nil(err)
	s.Contains(msg.String(), "status:finished")
}

func (s *suiteRelay) TestExec_9_QryStatus3() {
	addrCoins := &types.ReqRelayAddrCoins{
		Addr: addrTo,
		//Status: types.RelayOrderStatus_finished,
		Coins: []string{"BTC"},
	}

	var OrderIds [][]byte
	OrderIds = append(OrderIds, []byte(s.orderId))
	s.kvdb.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(OrderIds, nil).Once()
	param := types.Encode(addrCoins)
	msg, err := s.relay.Query("GetBuyRelayOrder", param)
	s.Nil(err)
	s.Contains(msg.String(), "status:finished")
}

func (s *suiteRelay) TestExec_9_QryStatus4() {
	addrCoins := &types.ReqRelayBtcHeaderHeightList{
		ReqHeight: 12,
		Counts:    2,
		Direction: 1,
	}

	var OrderIds [][]byte
	OrderIds = append(OrderIds, []byte(s.orderId))
	s.kvdb.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(OrderIds, nil).Once()
	param := types.Encode(addrCoins)
	msg, err := s.relay.Query("GetBTCHeaderList", param)
	s.Nil(err)
	//s.T().Log(msg)
	s.Contains(msg.String(), "heights:-1")
}

func (s *suiteRelay) TestExec_9_QryStatus5() {
	addrCoins := &types.ReqRelayQryBTCHeadHeight{
		BaseHeight: 10,
	}

	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Twice()
	param := types.Encode(addrCoins)
	msg, err := s.relay.Query("GetBTCHeaderCurHeight", param)
	s.Nil(err)
	//s.T().Log(msg)
	s.Contains(msg.String(), "curHeight:10 baseHeight:10")
}

func TestRunSuiteRelay(t *testing.T) {
	log := new(suiteRelay)
	suite.Run(t, log)
}

//////////////////////////////////////

type suiteBtcHeader struct {
	// Include our basic suite logic.
	suite.Suite
	db    *mocks.KV
	kvdb  *mocks.KVDB
	relay *relay
}

func (s *suiteBtcHeader) SetupSuite() {
	s.db = new(mocks.KV)
	s.kvdb = new(mocks.KVDB)
	//accDb, _ := db.NewGoMemDB("relayTestDb", "test", 128)
	relay := &relay{}
	relay.SetStateDB(s.db)
	relay.SetLocalDB(s.kvdb)
	relay.SetEnv(10, 100, 1)
	relay.SetIsFree(false)
	relay.SetApi(nil)
	relay.SetChild(relay)
	s.relay = relay

	s.Equal("relay", s.relay.GetName())

}

func (s *suiteBtcHeader) testExecBtcHeadLocal(tx *types.Transaction, receipt *types.Receipt, headers *types.BtcHeaders) {
	s.Equal(int32(types.ExecOk), receipt.Ty)
	rData := &types.ReceiptData{}
	rData.Ty = receipt.Ty
	rData.Logs = append(rData.Logs, receipt.Logs...)
	s.kvdb.On("Get", mock.Anything).Return([]byte{}, nil).Twice()
	set, err := s.relay.ExecLocal(tx, rData, 0)
	s.Nil(err)

	var kv []*types.KeyValue
	btc := newBtcStore(s.relay.GetLocalDB())
	for _, head := range headers.BtcHeader {
		val, err := btc.saveBlockHead(head)
		s.Nil(err)
		kv = append(kv, val...)

	}

	s.Subset(set.KV, kv)

}

func (s *suiteBtcHeader) testExecBtcHeadDelLocal(tx *types.Transaction, receipt *types.Receipt, headers *types.BtcHeaders) {
	s.Equal(int32(types.ExecOk), receipt.Ty)
	rData := &types.ReceiptData{}
	rData.Ty = receipt.Ty
	rData.Logs = append(rData.Logs, receipt.Logs...)
	s.kvdb.On("Get", mock.Anything).Return([]byte{}, nil).Twice()
	set, err := s.relay.ExecDelLocal(tx, rData, 0)
	s.Nil(err)

	var kv []*types.KeyValue
	btc := newBtcStore(s.relay.GetLocalDB())
	for _, head := range headers.BtcHeader {
		val, err := btc.delBlockHead(head)
		s.Nil(err)
		kv = append(kv, val...)

	}

	s.Subset(set.KV, kv)
}

//rcv btchead
func (s *suiteBtcHeader) TestSaveBtcHead_1() {
	head0 := &types.BtcHeader{
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
	head1 := &types.BtcHeader{
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

	head2 := &types.BtcHeader{
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

	headers := &types.BtcHeaders{}
	headers.BtcHeader = append(headers.BtcHeader, head0)
	headers.BtcHeader = append(headers.BtcHeader, head1)
	headers.BtcHeader = append(headers.BtcHeader, head2)

	sell := &types.RelayAction{
		Ty:    types.RelayActionRcvBTCHeaders,
		Value: &types.RelayAction_BtcHeaders{headers},
	}

	tx := &types.Transaction{}
	tx.Execer = types.ExecerRelay
	tx.To = address.ExecAddress(string(types.ExecerRelay))
	tx.Nonce = 2 //for different order id
	tx.Payload = types.Encode(sell)
	tx.Sign(types.SECP256K1, privFrom)

	s.relay.SetEnv(10, 1000, 1)

	s.db.On("Get", mock.Anything).Return(nil, types.ErrNotFound).Once()
	s.db.On("Set", mock.Anything, mock.Anything).Return(nil).Once()
	receipt, err := s.relay.Exec(tx, 0)
	s.Nil(err)

	s.testExecBtcHeadLocal(tx, receipt, headers)
	s.testExecBtcHeadDelLocal(tx, receipt, headers)

}

func TestRunSuiteBtcHeader(t *testing.T) {
	log := new(suiteBtcHeader)
	suite.Run(t, log)
}
