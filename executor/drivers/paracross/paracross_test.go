package paracross

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

	"gitlab.33.cn/chain33/chain33/blockchain"
	"gitlab.33.cn/chain33/chain33/common"
	"gitlab.33.cn/chain33/chain33/common/log"
	pt "gitlab.33.cn/chain33/chain33/types/executor/paracross"
)

// 构造一个4个节点的平行链数据， 进行测试
const (
	SignedType = types.SECP256K1
)

var (
	MainBlockHash10 = []byte("main block hash 10")
	MainBlockHeight = int64(10)
	CurHeight       = int64(10)
	Title           = string("test")
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
)

type ExecTestSuite struct {
	suite.Suite
	stateDB dbm.KV
	localDB *dbmock.KVDB
	api     *apimock.QueueProtocolAPI

	exec *Paracross
}

func makeNodeInfo(key, addr string) *types.ConfigItem {
	var item types.ConfigItem
	item.Key = key
	item.Addr = addr
	item.Ty = types.ConfigItemArrayConfig
	emptyValue := &types.ArrayConfig{make([]string, 0)}
	arr := types.ConfigItem_Arr{emptyValue}
	item.Value = &arr
	for _, n := range Nodes {
		item.GetArr().Value = append(item.GetArr().Value, string(n))
	}
	return &item
}

func init() {

}

func (suite *ExecTestSuite) SetupSuite() {
	log.SetFileLog(nil)
	log.SetLogLevel("debug")

	Init()
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

	// TODO, more fields
	// setup block
	blockDetail := &types.BlockDetail{
		Block: &types.Block{},
	}
	MainBlockHash10 = blockDetail.Block.Hash()

	// setup title nodes : len = 4
	nodeConfigKey := calcConfigNodesKey(Title)
	nodeValue := makeNodeInfo(Title, Title)
	suite.stateDB.Set(nodeConfigKey, types.Encode(nodeValue))
	value, err := suite.stateDB.Get(nodeConfigKey)
	if err != nil {
		suite.T().Error("get setup title failed", err)
		return
	}
	assert.Equal(suite.T(), value, types.Encode(nodeValue))

	// setup state title 'test' height is 9
	var titleStatus types.ParacrossStatus
	titleStatus.Title = Title
	titleStatus.Height = CurHeight - 1
	titleStatus.BlockHash = PerBlock
	saveTitle(suite.stateDB, calcTitleKey(Title), &titleStatus)

	// setup local block
	blockHeader10 := types.Header{
		Height: CurHeight,
		Hash:   MainBlockHash10,
	}
	header10key := blockchain.CalcHashToBlockHeaderKey(MainBlockHash10)
	suite.localDB.On("Get", header10key).Return(types.Encode(&blockHeader10), nil)


	// setup api
	hashes := &types.ReqHashes{[][]byte{MainBlockHash10}}
	suite.api.On("GetBlockByHashes", hashes).Return(
		&types.BlockDetails{
			Items: []*types.BlockDetail{blockDetail},
		}, nil)
	suite.api.On("GetBlockHash", &types.ReqInt{MainBlockHeight}).Return(
		&types.ReplyHash{MainBlockHash10}, nil)
}

func (suite *ExecTestSuite) TestSetup() {
	nodeConfigKey := calcConfigNodesKey(Title)
	suite.T().Log(string(nodeConfigKey))
	_, err := suite.stateDB.Get(nodeConfigKey)
	if err != nil {
		suite.T().Error("get setup title failed", err)
		return
	}
}

func commitOnce(suite *ExecTestSuite, privkeyStr string) (receipt *types.Receipt) {
	st1 := types.ParacrossNodeStatus{
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
	}
	tx := pt.CreateRawCommitTx(&st1)

	cr, err := crypto.New(types.GetSignatureTypeName(SignedType))
	if err != nil {
		suite.T().Error("TestExec", "new crypto failed", err)
		return
	}
	privBits, err := common.Hex2Bytes(privkeyStr[2:])
	if err != nil {
		suite.T().Error("TestExec", "Hex2Bytes privkey faiiled", err)
		return
	}
	privkey, err := cr.PrivKeyFromBytes(privBits)
	if err != nil {
		suite.T().Error("TestExec", "PrivKeyFromBytes failed", err)
		return
	}
	// make priv
	tx.Sign(types.SECP256K1, privkey)
	suite.T().Log(tx.From())
	receipt, err = suite.exec.Exec(tx, 0)
	suite.T().Log(receipt)
	assert.NotNil(suite.T(), receipt)
	assert.Nil(suite.T(), err)

	return
}

func checkCommitReceipt(suite *ExecTestSuite, receipt *types.Receipt, commitCnt int) {
	assert.Equal(suite.T(), receipt.Ty, int32(types.ExecOk))
	assert.Len(suite.T(), receipt.KV, 1)
	assert.Len(suite.T(), receipt.Logs, 1)

	key := calcTitleHeightKey(Title, TitleHeight)
	suite.T().Log("title height key", string(key))
	assert.Equal(suite.T(), key, receipt.KV[0].Key,
		"receipt not match", string(key), string(receipt.KV[0].Key))

	var titleHeight types.ParacrossHeightStatus
	err := types.Decode(receipt.KV[0].Value, &titleHeight)
	assert.Nil(suite.T(), err, "decode titleHeight failed")
	suite.T().Log("titleHeight", titleHeight)
	assert.Equal(suite.T(), int32(types.TyLogParacrossCommit), receipt.Logs[0].Ty)
	assert.Equal(suite.T(), int32(pt.ParacrossStatusCommiting), titleHeight.Status)
	assert.Equal(suite.T(), Title, titleHeight.Title)
	assert.Equal(suite.T(), commitCnt, len(titleHeight.Details.Addrs))
}

func checkDoneReceipt(suite *ExecTestSuite, receipt *types.Receipt, commitCnt int) {
	assert.Equal(suite.T(), receipt.Ty, int32(types.ExecOk))
	assert.Len(suite.T(), receipt.KV, 2)
	assert.Len(suite.T(), receipt.Logs, 2)

	key := calcTitleHeightKey(Title, TitleHeight)
	suite.T().Log("title height key", string(key))
	assert.Equal(suite.T(), key, receipt.KV[0].Key,
		"receipt not match", string(key), string(receipt.KV[0].Key))

	var titleHeight types.ParacrossHeightStatus
	err := types.Decode(receipt.KV[0].Value, &titleHeight)
	assert.Nil(suite.T(), err, "decode titleHeight failed")
	suite.T().Log("titleHeight", titleHeight)
	assert.Equal(suite.T(), int32(types.TyLogParacrossCommit), receipt.Logs[0].Ty)
	assert.Equal(suite.T(), int32(pt.ParacrossStatusCommiting), titleHeight.Status)
	assert.Equal(suite.T(), Title, titleHeight.Title)
	assert.Equal(suite.T(), commitCnt, len(titleHeight.Details.Addrs))

	keyTitle := calcTitleKey(Title)
	suite.T().Log("title key", string(keyTitle))
	assert.Equal(suite.T(), keyTitle, receipt.KV[1].Key,
		"receipt not match", string(keyTitle), string(receipt.KV[1].Key))

	var titleStat types.ParacrossStatus
	err = types.Decode(receipt.KV[1].Value, &titleStat)
	assert.Nil(suite.T(), err, "decode title failed")
	suite.T().Log("title", titleStat)
	assert.Equal(suite.T(), int32(types.TyLogParacrossDone), receipt.Logs[1].Ty)
	assert.Equal(suite.T(), int64(TitleHeight), titleStat.Height)
	assert.Equal(suite.T(), Title, titleStat.Title)
	assert.Equal(suite.T(), CurBlock, titleStat.BlockHash)
}

func checkRecordReceipt(suite *ExecTestSuite, receipt *types.Receipt, commitCnt int) {
	assert.Equal(suite.T(), receipt.Ty, int32(types.ExecOk))
	assert.Len(suite.T(), receipt.KV, 0)
	assert.Len(suite.T(), receipt.Logs, 1)

	var record types.ReceiptParacrossRecord
	err := types.Decode(receipt.Logs[0].Log, &record)
	assert.Nil(suite.T(), err)
	suite.T().Log("record", record)
	assert.Equal(suite.T(), int32(types.TyLogParacrossRecord), receipt.Logs[0].Ty)
	assert.Equal(suite.T(), Title, record.Status.Title)
	assert.Equal(suite.T(), int64(TitleHeight), record.Status.Height)
	assert.Equal(suite.T(), CurBlock, record.Status.BlockHash)
}

func (suite *ExecTestSuite) TestExec() {
	receipt := commitOnce(suite, PrivKeyA)
	checkCommitReceipt(suite, receipt, 1)

	receipt = commitOnce(suite, PrivKeyA)
	checkCommitReceipt(suite, receipt, 1)

	receipt = commitOnce(suite, PrivKeyB)
	checkCommitReceipt(suite, receipt, 2)

	receipt = commitOnce(suite, PrivKeyA)
	checkCommitReceipt(suite, receipt, 2)

	receipt = commitOnce(suite, PrivKeyC)
	checkDoneReceipt(suite, receipt, 3)

	receipt = commitOnce(suite, PrivKeyC)
	checkRecordReceipt(suite, receipt, 3)

	receipt = commitOnce(suite, PrivKeyD)
	checkRecordReceipt(suite, receipt, 4)
}

func TestExecSuite(t *testing.T) {
	suite.Run(t, new(ExecTestSuite))
}
