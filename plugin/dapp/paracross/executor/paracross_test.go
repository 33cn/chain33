package executor

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	//"github.com/stretchr/testify/mock"
	"testing"

	apimock "gitlab.33.cn/chain33/chain33/client/mocks"
	"gitlab.33.cn/chain33/chain33/common/crypto"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	dbmock "gitlab.33.cn/chain33/chain33/common/db/mocks"
	"gitlab.33.cn/chain33/chain33/types"

	"bytes"
	"math/rand"
	"time"

	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/common/log"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	mty "gitlab.33.cn/chain33/chain33/system/dapp/manage/types"
)

// 构造一个4个节点的平行链数据， 进行测试
const (
	SignedType = types.SECP256K1
)

var (
	MainBlockHash10 = []byte("main block hash 10")
	MainBlockHeight = int64(10)
	CurHeight       = int64(10)
	Title           = string("user.p.test.")
	TitleHeight     = int64(10)
	PerBlock        = []byte("block-hash-9")
	CurBlock        = []byte("block-hash-10")
	PerState        = []byte("state-hash-9")
	CurState        = []byte("state-hash-10")

	PrivKeyA = "0x6da92a632ab7deb67d38c0f6560bcfed28167998f6496db64c258d5e8393a81b" // 1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4
	PrivKeyB = "0x19c069234f9d3e61135fefbeb7791b149cdf6af536f26bebb310d4cd22c3fee4" // 1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR
	PrivKeyC = "0x7a80a1f75d7360c6123c32a78ecf978c1ac55636f87892df38d8b85a9aeff115" // 1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k
	PrivKeyD = "0xcacb1f5d51700aea07fca2246ab43b0917d70405c65edea9b5063d72eb5c6b71" // 1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs
	Nodes    = [][]byte{
		[]byte("1KSBd17H7ZK8iT37aJztFB22XGwsPTdwE4"),
		[]byte("1JRNjdEqp4LJ5fqycUBm9ayCKSeeskgMKR"),
		[]byte("1NLHPEcbTWWxxU3dGUZBhayjrCHD3psX7k"),
		[]byte("1MCftFynyvG2F4ED5mdHYgziDxx6vDrScs"),
	}

	TokenSymbol                = "X"
	MainBlockHeightForTransfer = int64(9)
)

type CommitTestSuite struct {
	suite.Suite
	stateDB dbm.KV
	localDB *dbmock.KVDB
	api     *apimock.QueueProtocolAPI

	exec *Paracross
}

func makeNodeInfo(key, addr string, cnt int) *types.ConfigItem {
	var item types.ConfigItem
	item.Key = key
	item.Addr = addr
	item.Ty = mty.ConfigItemArrayConfig
	emptyValue := &types.ArrayConfig{make([]string, 0)}
	arr := types.ConfigItem_Arr{emptyValue}
	item.Value = &arr
	for i, n := range Nodes {
		if i >= cnt {
			break
		}
		item.GetArr().Value = append(item.GetArr().Value, string(n))
	}
	return &item
}

func init() {
	log.SetFileLog(nil)
	log.SetLogLevel("debug")
	Init(pt.ParaX, nil)
}

func (suite *CommitTestSuite) SetupSuite() {

	suite.stateDB, _ = dbm.NewGoMemDB("state", "state", 1024)
	// memdb 不支持KVDB接口， 等测试完Exec ， 再扩展 memdb
	//suite.localDB, _ = dbm.NewGoMemDB("local", "local", 1024)
	suite.localDB = new(dbmock.KVDB)
	suite.api = new(apimock.QueueProtocolAPI)

	suite.exec = newParacross().(*Paracross)
	suite.exec.SetLocalDB(suite.localDB)
	suite.exec.SetStateDB(suite.stateDB)
	suite.exec.SetEnv(0, 0, 0)
	suite.exec.SetApi(suite.api)
	enableParacrossTransfer = false

	// TODO, more fields
	// setup block
	blockDetail := &types.BlockDetail{
		Block: &types.Block{},
	}
	MainBlockHash10 = blockDetail.Block.Hash()

	// setup title nodes : len = 4
	nodeConfigKey := calcConfigNodesKey(Title)
	nodeValue := makeNodeInfo(Title, Title, 4)
	suite.stateDB.Set(nodeConfigKey, types.Encode(nodeValue))
	value, err := suite.stateDB.Get(nodeConfigKey)
	if err != nil {
		suite.T().Error("get setup title failed", err)
		return
	}
	assert.Equal(suite.T(), value, types.Encode(nodeValue))

	// setup state title 'test' height is 9
	var titleStatus pt.ParacrossStatus
	titleStatus.Title = Title
	titleStatus.Height = CurHeight - 1
	titleStatus.BlockHash = PerBlock
	saveTitle(suite.stateDB, calcTitleKey(Title), &titleStatus)

	// setup api
	hashes := &types.ReqHashes{[][]byte{MainBlockHash10}}
	suite.api.On("GetBlockByHashes", hashes).Return(
		&types.BlockDetails{
			Items: []*types.BlockDetail{blockDetail},
		}, nil)
	suite.api.On("GetBlockHash", &types.ReqInt{MainBlockHeight}).Return(
		&types.ReplyHash{MainBlockHash10}, nil)
}

func (suite *CommitTestSuite) TestSetup() {
	nodeConfigKey := calcConfigNodesKey(Title)
	suite.T().Log(string(nodeConfigKey))
	_, err := suite.stateDB.Get(nodeConfigKey)
	if err != nil {
		suite.T().Error("get setup title failed", err)
		return
	}
}

func fillRawCommitTx(suite suite.Suite) (*types.Transaction, error) {
	st1 := pt.ParacrossNodeStatus{
		MainBlockHash10,
		MainBlockHeight,
		Title,
		TitleHeight,
		[]byte("block-hash-9"),
		[]byte("block-hash-10"),
		[]byte("state-hash-9"),
		[]byte("state-hash-10"),
		10,
		[]byte("abc"),
		[][]byte{},
		[]byte("abc"),
		[][]byte{},
	}
	tx, err := pt.CreateRawCommitTx4MainChain(&st1, pt.ParaX, 0)
	if err != nil {
		suite.T().Error("TestExec", "create tx failed", err)
	}
	return tx, err
}

func signTx(s suite.Suite, tx *types.Transaction, hexPrivKey string) (*types.Transaction, error) {
	signType := types.SECP256K1
	c, err := crypto.New(types.GetSignName("", signType))
	if err != nil {
		s.T().Error("TestExec", "new crypto failed", err)
		return tx, err
	}

	bytes, err := common.FromHex(hexPrivKey[:])
	if err != nil {
		s.T().Error("TestExec", "Hex2Bytes privkey faiiled", err)
		return tx, err
	}

	privKey, err := c.PrivKeyFromBytes(bytes)
	if err != nil {
		s.T().Error("TestExec", "PrivKeyFromBytes failed", err)
		return tx, err
	}

	tx.Sign(int32(signType), privKey)
	return tx, nil
}

func getPrivKey(s suite.Suite, hexPrivKey string) (crypto.PrivKey, error) {
	signType := types.SECP256K1
	c, err := crypto.New(types.GetSignName("", signType))
	if err != nil {
		s.T().Error("TestExec", "new crypto failed", err)
		return nil, err
	}

	bytes, err := common.FromHex(hexPrivKey[:])
	if err != nil {
		s.T().Error("TestExec", "Hex2Bytes privkey faiiled", err)
		return nil, err
	}

	privKey, err := c.PrivKeyFromBytes(bytes)
	if err != nil {
		s.T().Error("TestExec", "PrivKeyFromBytes failed", err)
		return nil, err
	}

	return privKey, nil
}

func commitOnce(suite *CommitTestSuite, privkeyStr string) (receipt *types.Receipt) {
	return commitOnceImpl(suite.Suite, suite.exec, privkeyStr)
}

func commitOnceImpl(suite suite.Suite, exec *Paracross, privkeyStr string) (receipt *types.Receipt) {
	tx, err := fillRawCommitTx(suite)
	tx, err = signTx(suite, tx, privkeyStr)

	suite.T().Log(tx.From())
	receipt, err = exec.Exec(tx, 0)
	suite.T().Log(receipt)
	assert.NotNil(suite.T(), receipt)
	assert.Nil(suite.T(), err)

	return
}

func checkCommitReceipt(suite *CommitTestSuite, receipt *types.Receipt, commitCnt int) {
	assert.Equal(suite.T(), receipt.Ty, int32(types.ExecOk))
	assert.Len(suite.T(), receipt.KV, 1)
	assert.Len(suite.T(), receipt.Logs, 1)

	key := calcTitleHeightKey(Title, TitleHeight)
	suite.T().Log("title height key", string(key))
	assert.Equal(suite.T(), key, receipt.KV[0].Key,
		"receipt not match", string(key), string(receipt.KV[0].Key))

	var titleHeight pt.ParacrossHeightStatus
	err := types.Decode(receipt.KV[0].Value, &titleHeight)
	assert.Nil(suite.T(), err, "decode titleHeight failed")
	suite.T().Log("titleHeight", titleHeight)
	assert.Equal(suite.T(), int32(pt.TyLogParacrossCommit), receipt.Logs[0].Ty)
	assert.Equal(suite.T(), int32(pt.ParacrossStatusCommiting), titleHeight.Status)
	assert.Equal(suite.T(), Title, titleHeight.Title)
	assert.Equal(suite.T(), commitCnt, len(titleHeight.Details.Addrs))
}

func checkDoneReceipt(suite suite.Suite, receipt *types.Receipt, commitCnt int) {
	assert.Equal(suite.T(), receipt.Ty, int32(types.ExecOk))
	assert.Len(suite.T(), receipt.KV, 2)
	assert.Len(suite.T(), receipt.Logs, 2)

	key := calcTitleHeightKey(Title, TitleHeight)
	suite.T().Log("title height key", string(key))
	assert.Equal(suite.T(), key, receipt.KV[0].Key,
		"receipt not match", string(key), string(receipt.KV[0].Key))

	var titleHeight pt.ParacrossHeightStatus
	err := types.Decode(receipt.KV[0].Value, &titleHeight)
	assert.Nil(suite.T(), err, "decode titleHeight failed")
	suite.T().Log("titleHeight", titleHeight)
	assert.Equal(suite.T(), int32(pt.TyLogParacrossCommit), receipt.Logs[0].Ty)
	assert.Equal(suite.T(), int32(pt.ParacrossStatusCommiting), titleHeight.Status)
	assert.Equal(suite.T(), Title, titleHeight.Title)
	assert.Equal(suite.T(), commitCnt, len(titleHeight.Details.Addrs))

	keyTitle := calcTitleKey(Title)
	suite.T().Log("title key", string(keyTitle), "receipt key", len(receipt.KV))
	assert.Equal(suite.T(), keyTitle, receipt.KV[1].Key,
		"receipt not match", string(keyTitle), string(receipt.KV[1].Key))

	var titleStat pt.ParacrossStatus
	err = types.Decode(receipt.KV[1].Value, &titleStat)
	assert.Nil(suite.T(), err, "decode title failed")
	suite.T().Log("title", titleStat)
	assert.Equal(suite.T(), int32(pt.TyLogParacrossCommitDone), receipt.Logs[1].Ty)
	assert.Equal(suite.T(), int64(TitleHeight), titleStat.Height)
	assert.Equal(suite.T(), Title, titleStat.Title)
	assert.Equal(suite.T(), CurBlock, titleStat.BlockHash)
}

func checkRecordReceipt(suite *CommitTestSuite, receipt *types.Receipt, commitCnt int) {
	assert.Equal(suite.T(), receipt.Ty, int32(types.ExecOk))
	assert.Len(suite.T(), receipt.KV, 0)
	assert.Len(suite.T(), receipt.Logs, 1)

	var record pt.ReceiptParacrossRecord
	err := types.Decode(receipt.Logs[0].Log, &record)
	assert.Nil(suite.T(), err)
	suite.T().Log("record", record)
	assert.Equal(suite.T(), int32(pt.TyLogParacrossCommitRecord), receipt.Logs[0].Ty)
	assert.Equal(suite.T(), Title, record.Status.Title)
	assert.Equal(suite.T(), int64(TitleHeight), record.Status.Height)
	assert.Equal(suite.T(), CurBlock, record.Status.BlockHash)
}

func (suite *CommitTestSuite) TestExec() {
	receipt := commitOnce(suite, PrivKeyA)
	checkCommitReceipt(suite, receipt, 1)

	receipt = commitOnce(suite, PrivKeyA)
	checkCommitReceipt(suite, receipt, 1)

	receipt = commitOnce(suite, PrivKeyB)
	checkCommitReceipt(suite, receipt, 2)

	receipt = commitOnce(suite, PrivKeyA)
	checkCommitReceipt(suite, receipt, 2)

	receipt = commitOnce(suite, PrivKeyC)
	checkDoneReceipt(suite.Suite, receipt, 3)

	receipt = commitOnce(suite, PrivKeyC)
	checkRecordReceipt(suite, receipt, 3)

	receipt = commitOnce(suite, PrivKeyD)
	checkRecordReceipt(suite, receipt, 4)
}

func TestCommitSuite(t *testing.T) {
	suite.Run(t, new(CommitTestSuite))
}

func TestGetTitle(t *testing.T) {
	exec := "p.user.guodun.token"
	titleExpect := []byte("p.user.guodun.")
	title, err := getTitleFrom([]byte(exec))
	if err != nil {
		t.Error("getTitleFrom", "failed", err)
		return
	}
	assert.Equal(t, titleExpect, title)
}

/*
func TestCrossLimits(t *testing.T) {
	stateDB, _ := dbm.NewGoMemDB("state", "state", 1024)
	localDB := new(dbmock.KVDB)
	api := new(apimock.QueueProtocolAPI)

	exec := newParacross().(*Paracross)
	exec.SetLocalDB(localDB)
	exec.SetStateDB(stateDB)
	exec.SetEnv(0, 0, 0)
	exec.SetApi(api)


	tx := &types.Transaction{Execer: []byte("p.user.test.paracross")}
	res := exec.CrossLimits(tx, 1)
	assert.True(t, res)

}
*/

type VoteTestSuite struct {
	suite.Suite
	exec *Paracross
}

func (suite *VoteTestSuite) SetupSuite() {
	types.Init(Title, nil)
	suite.exec = newParacross().(*Paracross)
}

func (s *VoteTestSuite) TestVoteTx() {
	status := &pt.ParacrossNodeStatus{
		MainBlockHash:   MainBlockHash10,
		MainBlockHeight: MainBlockHeight,
		PreBlockHash:    PerBlock,
		Height:          CurHeight,
		Title:           Title,
	}
	tx, err := s.createVoteTx(status, PrivKeyA)
	s.Nil(err)
	tx1, err := createAssetTransferTx(s.Suite, PrivKeyA, nil)
	s.Nil(err)
	tx2, err := createAssetTransferTx(s.Suite, PrivKeyB, nil)
	s.Nil(err)
	tx3, err := createCrossMainTx([]byte("toA"))
	s.Nil(err)
	tx4, err := createCrossParaTx(s.Suite, []byte("toB"))
	s.Nil(err)
	tx34 := []*types.Transaction{tx3, tx4}
	txGroup34, err := createTxsGroup(s.Suite, tx34)
	s.Nil(err)

	tx5, err := createCrossParaTx(s.Suite, nil)
	s.Nil(err)
	tx6, err := createCrossParaTx(s.Suite, nil)
	s.Nil(err)
	tx56 := []*types.Transaction{tx5, tx6}
	txGroup56, err := createTxsGroup(s.Suite, tx56)
	s.Nil(err)

	tx7, err := createAssetTransferTx(s.Suite, PrivKeyC, nil)
	s.Nil(err)
	txs := []*types.Transaction{tx, tx1, tx2}
	txs = append(txs, txGroup34...)
	txs = append(txs, txGroup56...)
	txs = append(txs, tx7)
	s.exec.SetTxs(txs)

	//for i,tx := range txs{
	//	s.T().Log("tx exec name","i",i,"name",string(tx.Execer))
	//}

	receipt0, err := s.exec.Exec(tx, 0)
	s.Nil(err)
	recpt0 := &types.ReceiptData{Ty: receipt0.Ty, Logs: receipt0.Logs}
	recpt1 := &types.ReceiptData{Ty: types.ExecOk}
	recpt2 := &types.ReceiptData{Ty: types.ExecErr}
	recpt3 := &types.ReceiptData{Ty: types.ExecOk}
	recpt4 := &types.ReceiptData{Ty: types.ExecOk}
	recpt5 := &types.ReceiptData{Ty: types.ExecPack}
	recpt6 := &types.ReceiptData{Ty: types.ExecPack}
	recpt7 := &types.ReceiptData{Ty: types.ExecOk}
	receipts := []*types.ReceiptData{recpt0, recpt1, recpt2, recpt3, recpt4, recpt5, recpt6, recpt7}
	s.exec.SetReceipt(receipts)
	set, err := s.exec.ExecLocal(tx, recpt0, 0)
	s.Nil(err)
	key := pt.CalcMinerHeightKey(status.Title, status.Height)
	for _, kv := range set.KV {
		//s.T().Log(string(kv.GetKey()))
		if bytes.Equal(key, kv.Key) {
			var rst pt.ParacrossNodeStatus
			types.Decode(kv.GetValue(), &rst)
			s.Equal([]uint8([]byte{0x25}), rst.TxResult)
			s.Equal([]uint8([]byte{0x4d}), rst.CrossTxResult)
			s.Equal(6, len(rst.TxHashs))
			s.Equal(7, len(rst.CrossTxHashs))
			break
		}
	}
}

func (s *VoteTestSuite) createVoteTx(status *pt.ParacrossNodeStatus, privFrom string) (*types.Transaction, error) {
	tx, err := pt.CreateRawMinerTx(status)
	assert.Nil(s.T(), err, "create asset transfer failed")
	if err != nil {
		return nil, err
	}

	tx, err = signTx(s.Suite, tx, privFrom)
	assert.Nil(s.T(), err, "sign asset transfer failed")
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func createCrossMainTx(to []byte) (*types.Transaction, error) {
	param := types.CreateTx{
		To:          string(to),
		Amount:      Amount,
		Fee:         0,
		Note:        "test asset transfer",
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    pt.ParaX,
	}
	transfer := &pt.ParacrossAction{}
	v := &pt.ParacrossAction_AssetTransfer{AssetTransfer: &types.AssetsTransfer{
		Amount: param.Amount, Note: param.GetNote(), To: param.GetTo()}}
	transfer.Value = v
	transfer.Ty = pt.ParacrossActionAssetTransfer

	tx := &types.Transaction{
		Execer:  []byte(param.GetExecName()),
		Payload: types.Encode(transfer),
		To:      address.ExecAddress(param.GetExecName()),
		Fee:     param.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
	}

	return tx, nil
}

func createCrossParaTx(s suite.Suite, to []byte) (*types.Transaction, error) {
	param := types.CreateTx{
		To:          string(to),
		Amount:      Amount,
		Fee:         0,
		Note:        "test asset transfer",
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    Title + pt.ParaX,
	}
	tx, err := pt.CreateRawAssetTransferTx(&param)
	assert.Nil(s.T(), err, "create asset transfer failed")
	if err != nil {
		return nil, err
	}

	//tx, err = signTx(s, tx, privFrom)
	//assert.Nil(s.T(), err, "sign asset transfer failed")
	//if err != nil {
	//	return nil, err
	//}

	return tx, nil
}

func createTxsGroup(s suite.Suite, txs []*types.Transaction) ([]*types.Transaction, error) {

	group, err := types.CreateTxGroup(txs)
	if err != nil {
		return nil, err
	}
	err = group.Check(0, types.GInt("MinFee"))
	if err != nil {
		return nil, err
	}
	privKey, err := getPrivKey(s, PrivKeyA)
	for i := range group.Txs {
		group.SignN(i, int32(types.SECP256K1), privKey)
	}

	return group.Txs, nil
}

func TestVoteSuite(t *testing.T) {
	suite.Run(t, new(VoteTestSuite))
}
