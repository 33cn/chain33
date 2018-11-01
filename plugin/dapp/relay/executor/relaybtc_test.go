package executor

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gitlab.33.cn/chain33/chain33/common/db/mocks"
	ty "gitlab.33.cn/chain33/chain33/plugin/dapp/relay/types"
	"gitlab.33.cn/chain33/chain33/types"
)

type suiteBtcStore struct {
	// Include our basic suite logic.
	suite.Suite
	kvdb *mocks.KVDB
	btc  *btcStore
}

func TestRunSuiteBtc(t *testing.T) {
	btc := new(suiteBtcStore)
	suite.Run(t, btc)
}

func (s *suiteBtcStore) SetupSuite() {
	s.kvdb = new(mocks.KVDB)
	s.btc = newBtcStore(s.kvdb)
}

func (s *suiteBtcStore) TestGetBtcHeadHeightFromDb() {
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	val, _ := s.btc.getBtcHeadHeightFromDb([]byte("key"))
	s.Assert().Equal(val, int64(10))
}

func (s *suiteBtcStore) TestGetLastBtcHeadHeight() {
	heightBytes := types.Encode(&types.Int64{int64(10)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	val, _ := s.btc.getLastBtcHeadHeight()
	s.Assert().Equal(val, int64(10))
}

func (s *suiteBtcStore) TestGetBtcHeadByHeight() {
	head := &ty.BtcHeader{}

	header := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(header, nil).Once()
	val, _ := s.btc.getBtcHeadByHeight(10)
	s.Assert().Equal(val, head)

}

func (s *suiteBtcStore) TestGetLastBtcHead() {
	heightBytes := types.Encode(&types.Int64{int64(10)})
	head := &ty.BtcHeader{}

	header := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once().On("Get", mock.Anything).Return(header, nil).Once()
	val, err := s.btc.getLastBtcHead()
	s.Assert().Nil(err)
	s.Assert().Equal(val, head)
}

func (s *suiteBtcStore) TestSaveBlockHead() {
	var kv []*types.KeyValue
	var head = &ty.BtcHeader{
		Version:      1,
		Hash:         "00000000d1145790a8694403d4063f323d499e655c83426834d4ce2f8dd4a2ee",
		MerkleRoot:   "7dac2c5666815c17a3b36427de37bb9d2e2c5ccec3f8633eb91a4205cb4c10ff",
		PreviousHash: "000000002a22cfee1f2c846adbd12b3e183d4f97683f85dad08a79780a84bd55",
		Bits:         486604799,
		Nonce:        1889418792,
		Time:         1231731025,
		Height:       2,
	}
	val, err := proto.Marshal(head)
	key := calcBtcHeaderKeyHash(head.Hash)
	kv = append(kv, &types.KeyValue{key, val})
	key = calcBtcHeaderKeyHeight(int64(head.Height))
	kv = append(kv, &types.KeyValue{key, val})
	key = calcBtcHeaderKeyHeightList(int64(head.Height))
	heightBytes := types.Encode(&types.Int64{int64(head.Height)})
	kv = append(kv, &types.KeyValue{key, heightBytes})

	res, err := s.btc.saveBlockHead(head)
	s.Nil(err)
	s.Equal(kv, res)
}

func (s *suiteBtcStore) TestSaveBlockLastHead() {
	var kv []*types.KeyValue

	lastHead := &ty.ReceiptRelayRcvBTCHeaders{
		LastHeight:     100,
		NewHeight:      200,
		LastBaseHeight: 10,
		NewBaseHeight:  150,
	}

	heightBytes := types.Encode(&types.Int64{int64(lastHead.NewHeight)})
	key := relayBTCHeaderLastHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	heightBytes = types.Encode(&types.Int64{int64(lastHead.NewBaseHeight)})
	key = relayBTCHeaderBaseHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	res, err := s.btc.saveBlockLastHead(lastHead)
	s.Nil(err)
	s.Equal(kv, res)
}

func (s *suiteBtcStore) TestDelBlockHead() {
	var kv []*types.KeyValue
	var head = &ty.BtcHeader{
		Version:      1,
		Hash:         "00000000d1145790a8694403d4063f323d499e655c83426834d4ce2f8dd4a2ee",
		MerkleRoot:   "7dac2c5666815c17a3b36427de37bb9d2e2c5ccec3f8633eb91a4205cb4c10ff",
		PreviousHash: "000000002a22cfee1f2c846adbd12b3e183d4f97683f85dad08a79780a84bd55",
		Bits:         486604799,
		Nonce:        1889418792,
		Time:         1231731025,
		Height:       2,
	}

	key := calcBtcHeaderKeyHash(head.Hash)
	kv = append(kv, &types.KeyValue{key, nil})
	// height:header
	key = calcBtcHeaderKeyHeight(int64(head.Height))
	kv = append(kv, &types.KeyValue{key, nil})

	// prefix-height:height
	key = calcBtcHeaderKeyHeightList(int64(head.Height))
	kv = append(kv, &types.KeyValue{key, nil})

	res, err := s.btc.delBlockHead(head)
	s.Nil(err)
	s.Equal(kv, res)
}

func (s *suiteBtcStore) TestDelBlockLastHead() {
	var kv []*types.KeyValue

	lastHead := &ty.ReceiptRelayRcvBTCHeaders{
		LastHeight:     100,
		NewHeight:      200,
		LastBaseHeight: 10,
		NewBaseHeight:  150,
	}

	heightBytes := types.Encode(&types.Int64{int64(lastHead.LastHeight)})
	key := relayBTCHeaderLastHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	heightBytes = types.Encode(&types.Int64{int64(lastHead.LastBaseHeight)})
	key = relayBTCHeaderBaseHeight
	kv = append(kv, &types.KeyValue{key, heightBytes})

	res, err := s.btc.delBlockLastHead(lastHead)
	s.Nil(err)
	s.Equal(kv, res)
}

func (s *suiteBtcStore) TestGetBtcCurHeight() {
	s.kvdb.On("Get", mock.Anything).Return(nil, types.ErrNotFound).Once().On("Get", mock.Anything).Return(nil, types.ErrNotFound).Once()

	rep, err := s.btc.getBtcCurHeight(nil)
	s.Nil(err)
	s.Equal(rep, &ty.ReplayRelayQryBTCHeadHeight{-1, -1})
}

func (s *suiteBtcStore) TestGetMerkleRootFromHeader() {
	var head = &ty.BtcHeader{
		Version:      1,
		Hash:         "00000000d1145790a8694403d4063f323d499e655c83426834d4ce2f8dd4a2ee",
		MerkleRoot:   "7dac2c5666815c17a3b36427de37bb9d2e2c5ccec3f8633eb91a4205cb4c10ff",
		PreviousHash: "000000002a22cfee1f2c846adbd12b3e183d4f97683f85dad08a79780a84bd55",
		Bits:         486604799,
		Nonce:        1889418792,
		Time:         1231731025,
		Height:       2,
	}

	head_enc := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(head_enc, nil).Once()
	res, err := s.btc.getMerkleRootFromHeader(head.Hash)
	s.Nil(err)
	s.Equal(head.MerkleRoot, res)
}

func (s *suiteBtcStore) TestVerifyBtcTx() {
	order := &ty.RelayOrder{
		CoinAddr:    "1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT",
		CoinAmount:  29900000,
		AcceptTime:  100,
		ConfirmTime: 200,
	}
	vout := &ty.Vout{
		Address: "1Am9UTGfdnxabvcywYG2hvzr6qK8T3oUZT",
		Value:   29900000,
	}
	transaction := &ty.BtcTransaction{
		Vout:        []*ty.Vout{vout},
		Time:        150,
		BlockHeight: 1000,
		Hash:        "6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	}

	//the 100000 height block in BTC
	//rootHash := "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766"

	//txarr := []string{"8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87",
	//					"fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
	//					"6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	//					"e9a66845e05d5abc0ad04ec80f774a7e585c6e8db975962d069a522137b80c1d"}
	//the 3rd tx's branch
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
	verify := &ty.RelayVerify{
		Tx:  transaction,
		Spv: spv,
	}

	heightBytes := types.Encode(&types.Int64{int64(1006)})
	s.kvdb.On("Get", mock.Anything).Return(heightBytes, nil).Once()
	var head = &ty.BtcHeader{
		Version:    1,
		MerkleRoot: "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
	}
	headEnc := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(headEnc, nil).Once()
	err := s.btc.verifyBtcTx(verify, order)
	s.Nil(err)
}

func (s *suiteBtcStore) TestVerifyCmdBtcTx() {

	//the 100000 height block in BTC
	//rootHash := "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766"

	//txarr := []string{"8c14f0db3df150123e6f3dbbf30f8b955a8249b62ac1d1ff16284aefa3d06d87",
	//					"fff2525b8931402dd09222c50775608f75787bd2b87e56995a7bdd30f79702c4",
	//					"6359f0868171b1d194cbee1af2f16ea598ae8fad666d9b012c8ed2b79a236ec4",
	//					"e9a66845e05d5abc0ad04ec80f774a7e585c6e8db975962d069a522137b80c1d"}

	verify := &ty.RelayVerifyCli{
		RawTx:      "0100000001c33ebff2a709f13d9f9a7569ab16a32786af7d7e2de09265e41c61d078294ecf010000008a4730440220032d30df5ee6f57fa46cddb5eb8d0d9fe8de6b342d27942ae90a3231e0ba333e02203deee8060fdc70230a7f5b4ad7d7bc3e628cbe219a886b84269eaeb81e26b4fe014104ae31c31bf91278d99b8377a35bbce5b27d9fff15456839e919453fc7b3f721f0ba403ff96c9deeb680e5fd341c0fc3a7b90da4631ee39560639db462e9cb850fffffffff0240420f00000000001976a914b0dcbf97eabf4404e31d952477ce822dadbe7e1088acc060d211000000001976a9146b1281eec25ab4e1e0793ff4e08ab1abb3409cd988ac00000000",
		TxIndex:    2,
		MerkBranch: "e9a66845e05d5abc0ad04ec80f774a7e585c6e8db975962d069a522137b80c1d-ccdafb73d8dcd0173d5d5c3c9a0770d0b3953db889dab99ef05b1907518cb815",
		BlockHash:  "000000000003ba27aa200b1cecaad478d2b00432346c3f1f3986da1afd33e506",
	}

	var head = &ty.BtcHeader{
		Version:    1,
		MerkleRoot: "f3e94742aca4b5ef85488dc37c06c3282295ffec960994b2c0d5ac2a25a95766",
	}
	headEnc := types.Encode(head)
	s.kvdb.On("Get", mock.Anything).Return(headEnc, nil).Once()

	err := s.btc.verifyCmdBtcTx(verify)
	s.Nil(err)
}

func (s *suiteBtcStore) TestGetHeadHeightList() {
	req := &ty.ReqRelayBtcHeaderHeightList{
		Counts:    1,
		Direction: 0,
	}
	var replay ty.ReplyRelayBtcHeadHeightList
	heightArry := make([][]byte, 10)
	for i := 0; i < 10; i++ {
		height := int64(1000 + i)
		heightBytes := types.Encode(&types.Int64{height})
		heightArry[i] = heightBytes
		replay.Heights = append(replay.Heights, height)
	}

	s.kvdb.On("List", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(heightArry, nil).Once()
	val, err := s.btc.getHeadHeightList(req)
	s.Nil(err)
	s.Equal(&replay, val)
}

func (s *suiteBtcStore) TestVerifyBlockHeader() {
	var head = &ty.BtcHeader{
		Version:      1,
		Hash:         "00000000d1145790a8694403d4063f323d499e655c83426834d4ce2f8dd4a2ee",
		MerkleRoot:   "7dac2c5666815c17a3b36427de37bb9d2e2c5ccec3f8633eb91a4205cb4c10ff",
		PreviousHash: "000000002a22cfee1f2c846adbd12b3e183d4f97683f85dad08a79780a84bd55",
		Bits:         486604799,
		Nonce:        1889418792,
		Time:         1231731025,
		Height:       2,
	}

	var lastHead = &ty.BtcHeader{
		Version: 1,
		Hash:    "000000002a22cfee1f2c846adbd12b3e183d4f97683f85dad08a79780a84bd55",
		Height:  1,
		Bits:    486604799,
	}
	var preHead = &ty.RelayLastRcvBtcHeader{
		Header:     lastHead,
		BaseHeight: 1,
	}

	err := verifyBlockHeader(head, preHead, s.kvdb)
	s.Nil(err)
}

func (s *suiteBtcStore) TestVerifyBlockHeader_2() {

	lastHead := &ty.BtcHeader{
		Version:      1,
		PreviousHash: "000000003ae696c44274e40817d4acaf40c1ff1853411d4f0573421caf5faa07",
		MerkleRoot:   "f1b2e16f74ee0e90cad3dc2e5be4806ff0581ed50a9cb3dfeee591dac76b17a7",
		Time:         1266189979,
		Bits:         486575299,
		Nonce:        126672526,
		Hash:         "000000000683a474ef810000fd22f0edde4cf33ae76ae506b220e57aeeafeaa4",
		Height:       40318,
	}
	var preHead = &ty.RelayLastRcvBtcHeader{
		Header:     lastHead,
		BaseHeight: 1,
	}
	head := &ty.BtcHeader{
		Version:      1,
		PreviousHash: "000000000683a474ef810000fd22f0edde4cf33ae76ae506b220e57aeeafeaa4",
		MerkleRoot:   "b4d736ca74838036ebd19b085c3eeb9ffec2307f6452347cdd8ddaa249686f39",
		Time:         1266190073,
		Bits:         386575299, //changed the bits from 4xxx->3xxx
		Nonce:        32196448,
		Hash:         "000000008135b689ad1557d4e148a8b9e58e2c4a67240fc87962abb69710231a",
		Height:       40319,
	}

	err := verifyBlockHeader(head, preHead, s.kvdb)
	s.Equal(ty.ErrRelayBtcHeadNewBitsErr, err)

}

func (s *suiteBtcStore) TestCalcNextRequiredDifficulty_1() {

	firstSelfHash, _ := chainhash.NewHashFromStr("0000000015bb50096055846954f7120e30d6aa2bd5ab8d4a4055ceacc853328a")
	lastSelfHash, _ := chainhash.NewHashFromStr("000000008135b689ad1557d4e148a8b9e58e2c4a67240fc87962abb69710231a")
	newSelfHash, _ := chainhash.NewHashFromStr("0000000045861e169b5a961b7034f8de9e98022e7a39100dde3ae3ea240d7245")
	firstHead := &ty.BtcHeader{
		Version:      1,
		PreviousHash: "000000008dde642fb80481bb5e1671cb04c6716de5b7f783aa3388456d5c8a85",
		MerkleRoot:   "012e04850c40b1beee7d9df6e0cc3afa8f13fa8deb0eea92bbfdb52910c108e4",
		Time:         1265319794,
		Bits:         486575299,
		Nonce:        3582704905,
		Hash:         "0000000015bb50096055846954f7120e30d6aa2bd5ab8d4a4055ceacc853328a",
		Height:       38304,
	}

	lastHead := &ty.BtcHeader{
		Version:      1,
		PreviousHash: "000000000683a474ef810000fd22f0edde4cf33ae76ae506b220e57aeeafeaa4",
		MerkleRoot:   "b4d736ca74838036ebd19b085c3eeb9ffec2307f6452347cdd8ddaa249686f39",
		Time:         1266190073,
		Bits:         486575299,
		Nonce:        32196448,
		Hash:         "000000008135b689ad1557d4e148a8b9e58e2c4a67240fc87962abb69710231a",
		Height:       40319,
	}

	newHead := &ty.BtcHeader{
		Version:      1,
		PreviousHash: "000000008135b689ad1557d4e148a8b9e58e2c4a67240fc87962abb69710231a",
		MerkleRoot:   "a86b3c149f204d4cb47c67bf9bfeea2719df101dd6e6fc3f0e60d86efeba22a8",
		Time:         1266191579,
		Bits:         476399191,
		Nonce:        404824782,
		Hash:         "0000000045861e169b5a961b7034f8de9e98022e7a39100dde3ae3ea240d7245",
		Height:       40320,
	}

	firstWireHead, _ := btcWireHeader(firstHead)
	lastWireHead, _ := btcWireHeader(lastHead)
	newWireHead, _ := btcWireHeader(newHead)
	s.Equal(*firstSelfHash, firstWireHead.BlockHash())
	s.Equal(*lastSelfHash, lastWireHead.BlockHash())
	s.Equal(*newSelfHash, newWireHead.BlockHash())

	firstHeadEncode := types.Encode(firstHead)
	s.kvdb.On("Get", mock.Anything).Return(firstHeadEncode, nil).Once()
	newbits, _ := calcNextRequiredDifficulty(lastHead, s.kvdb)
	s.Equal(newHead.Bits, newbits)

}

func (s *suiteBtcStore) TestCalcNextRequiredDifficulty_2() {

	firstSelfHash, _ := chainhash.NewHashFromStr("0000000000000000006cd44d7a940c79f94c7c272d159ba19feb15891aa1ea54")
	lastSelfHash, _ := chainhash.NewHashFromStr("0000000000000000006a8957cbd52c2038861514f106f7f9f76392d5cb83fd4c")
	firstHead := &ty.BtcHeader{
		Version:      536870912,
		PreviousHash: "000000000000000000720da39f66f29337b9a29223e1ce05fd5ee57bb72a9223",
		MerkleRoot:   "6195fe0cded5aeb07c8d36826758343778ccd81e4285bba0f76e35e8549ab93c",
		Time:         1515827554,
		Bits:         394155916,
		Nonce:        3750147913,
		Hash:         "0000000000000000006cd44d7a940c79f94c7c272d159ba19feb15891aa1ea54",
		Height:       504000,
	}

	lastHead := &ty.BtcHeader{
		Version:      536870912,
		PreviousHash: "00000000000000000072aeab8610166cca1e3d4d8407d5a8edff5bfa78f6b149",
		MerkleRoot:   "416100b7e19d37af6a777ec82b0691ceef2f70da5050d8ed1afe5981ba5b6c17",
		Time:         1516862792,
		Bits:         394155916,
		Nonce:        1192182831,
		Hash:         "0000000000000000006a8957cbd52c2038861514f106f7f9f76392d5cb83fd4c",
		Height:       506015,
	}

	newHead := &ty.BtcHeader{
		Version:      536870912,
		PreviousHash: "0000000000000000006a8957cbd52c2038861514f106f7f9f76392d5cb83fd4c",
		MerkleRoot:   "82b19050016e068a2f69b34d66165d2fc9372840609d27c383c9da14f2537dba",
		Time:         1516862900,
		Bits:         392962374,
		Nonce:        4220492261,
		Hash:         "000000000000000000258fe79c8b891a7cecdf96196c6c9b2e609318c7ad95c9",
		Height:       506016,
	}

	firstWireHead, _ := btcWireHeader(firstHead)
	lastWireHead, _ := btcWireHeader(lastHead)
	s.Equal(*firstSelfHash, firstWireHead.BlockHash())
	s.Equal(*lastSelfHash, lastWireHead.BlockHash())

	firstHeadEncode := types.Encode(firstHead)
	s.kvdb.On("Get", mock.Anything).Return(firstHeadEncode, nil).Once()
	newbits, _ := calcNextRequiredDifficulty(lastHead, s.kvdb)
	s.Equal(newHead.Bits, newbits)

}
