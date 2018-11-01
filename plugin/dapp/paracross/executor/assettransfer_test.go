package executor

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	//"github.com/stretchr/testify/mock"
	"testing"

	"gitlab.33.cn/chain33/chain33/account"
	apimock "gitlab.33.cn/chain33/chain33/client/mocks"
	"gitlab.33.cn/chain33/chain33/common/address"
	dbm "gitlab.33.cn/chain33/chain33/common/db"
	dbmock "gitlab.33.cn/chain33/chain33/common/db/mocks"
	pt "gitlab.33.cn/chain33/chain33/plugin/dapp/paracross/types"
	"gitlab.33.cn/chain33/chain33/types"
)

// para-exec addr on main 1HPkPopVe3ERfvaAgedDtJQ792taZFEHCe
// para-exec addr on para 16zsMh7mvNDKPG6E9NVrPhw6zL93gWsTpR

var (
	Amount = int64(1 * types.Coin)
)

// 构建跨链交易, 用1个节点即可， 不测试共识
//    assetTransfer
//	 分别测试在主链和平行链的情况

type AssetTransferTestSuite struct {
	suite.Suite
	stateDB dbm.KV
	localDB *dbmock.KVDB
	api     *apimock.QueueProtocolAPI

	exec *Paracross
}

func TestAssetTransfer(t *testing.T) {
	suite.Run(t, new(AssetTransferTestSuite))
}

func (suite *AssetTransferTestSuite) SetupTest() {
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
	enableParacrossTransfer = true

	// setup block
	blockDetail := &types.BlockDetail{
		Block: &types.Block{},
	}
	MainBlockHash10 = blockDetail.Block.Hash()

	// setup title nodes : len = 1
	nodeConfigKey := calcConfigNodesKey(Title)
	nodeValue := makeNodeInfo(Title, Title, 1)
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

func (suite *AssetTransferTestSuite) TestExecTransferNobalance() {
	types.Init("test", nil)
	toB := Nodes[1]
	tx, err := createAssetTransferTx(suite.Suite, PrivKeyD, toB)
	if err != nil {
		suite.T().Error("TestExecTransfer", "createTxGroup", err)
		return
	}

	_, err = suite.exec.Exec(tx, 1)
	if err != types.ErrNoBalance {
		suite.T().Error("Exec Transfer", err)
		return
	}
}

func (suite *AssetTransferTestSuite) TestExecTransfer() {
	types.Init("test", nil)
	toB := Nodes[1]

	total := 1000 * types.Coin
	accountA := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	acc := account.NewCoinsAccount()
	acc.SetDB(suite.stateDB)
	addrMain := address.ExecAddress(pt.ParaX)
	addrPara := address.ExecAddress(Title + pt.ParaX)

	acc.SaveExecAccount(addrMain, &accountA)

	tx, err := createAssetTransferTx(suite.Suite, PrivKeyA, toB)
	if err != nil {
		suite.T().Error("TestExecTransfer", "createTxGroup", err)
		return
	}
	suite.T().Log(string(tx.Execer))
	receipt, err := suite.exec.Exec(tx, 1)
	if err != nil {
		suite.T().Error("Exec Transfer", err)
		return
	}
	for _, kv := range receipt.KV {
		var v types.Account
		err = types.Decode(kv.Value, &v)
		if err != nil {
			// skip, only check frozen
			continue
		}
		suite.T().Log(string(kv.Key), v)
	}
	suite.T().Log("para-exec addr on main", addrMain)
	suite.T().Log("para-exec addr on para", addrPara)
	suite.T().Log("para-exec addr for A account", accountA.Addr)
	accTest := acc.LoadExecAccount(addrPara, addrMain)
	assert.Equal(suite.T(), Amount, accTest.Balance)

	resultA := acc.LoadExecAccount(string(Nodes[0]), addrMain)
	assert.Equal(suite.T(), total-Amount, resultA.Balance)
}

func (suite *AssetTransferTestSuite) TestExecTransferInPara() {
	types.Init(Title, nil)
	toB := Nodes[1]

	tx, err := createAssetTransferTx(suite.Suite, PrivKeyA, toB)
	if err != nil {
		suite.T().Error("TestExecTransfer", "createTxGroup", err)
		return
	}

	receipt, err := suite.exec.Exec(tx, 1)
	if err != nil {
		suite.T().Error("Exec Transfer", err)
		return
	}
	for _, kv := range receipt.KV {
		var v types.Account
		err = types.Decode(kv.Value, &v)
		if err != nil {
			// skip, only check frozen
			continue
		}
		suite.T().Log(string(kv.Key), v)
	}

	acc, _ := NewParaAccount(Title, "coins", "bty", suite.stateDB)
	resultB := acc.LoadAccount(string(toB))
	assert.Equal(suite.T(), Amount, resultB.Balance)
}

func createAssetTransferTx(s suite.Suite, privFrom string, to []byte) (*types.Transaction, error) {
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

	tx, err = signTx(s, tx, privFrom)
	assert.Nil(s.T(), err, "sign asset transfer failed")
	if err != nil {
		return nil, err
	}

	return tx, nil
}

const TestSymbol = "TEST"

func (suite *AssetTransferTestSuite) TestExecTransferToken() {
	types.Init("test", nil)
	toB := Nodes[1]

	total := 1000 * types.Coin
	accountA := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	acc, _ := account.NewAccountDB("token", TestSymbol, suite.stateDB)
	addrMain := address.ExecAddress(pt.ParaX)
	addrPara := address.ExecAddress(Title + pt.ParaX)

	acc.SaveExecAccount(addrMain, &accountA)

	tx, err := createAssetTransferTokenTx(suite.Suite, PrivKeyA, toB)
	if err != nil {
		suite.T().Error("TestExecTransfer", "createTxGroup", err)
		return
	}
	suite.T().Log(string(tx.Execer))
	receipt, err := suite.exec.Exec(tx, 1)
	if err != nil {
		suite.T().Error("Exec Transfer", err)
		return
	}
	for _, kv := range receipt.KV {
		var v types.Account
		err = types.Decode(kv.Value, &v)
		if err != nil {
			// skip, only check frozen
			continue
		}
		suite.T().Log(string(kv.Key), v)
	}
	suite.T().Log("para-exec addr on main", addrMain)
	suite.T().Log("para-exec addr on para", addrPara)
	suite.T().Log("para-exec addr for A account", accountA.Addr)
	accTest := acc.LoadExecAccount(addrPara, addrMain)
	assert.Equal(suite.T(), Amount, accTest.Balance)

	resultA := acc.LoadExecAccount(string(Nodes[0]), addrMain)
	assert.Equal(suite.T(), total-Amount, resultA.Balance)
}

func (suite *AssetTransferTestSuite) TestExecTransferTokenInPara() {
	types.Init(Title, nil)
	toB := Nodes[1]

	tx, err := createAssetTransferTokenTx(suite.Suite, PrivKeyA, toB)
	if err != nil {
		suite.T().Error("TestExecTransfer", "createTxGroup", err)
		return
	}

	receipt, err := suite.exec.Exec(tx, 1)
	if err != nil {
		suite.T().Error("Exec Transfer", err)
		return
	}
	for _, kv := range receipt.KV {
		var v types.Account
		err = types.Decode(kv.Value, &v)
		if err != nil {
			// skip, only check frozen
			continue
		}
		suite.T().Log(string(kv.Key), v)
	}

	acc, _ := NewParaAccount(Title, "token", TestSymbol, suite.stateDB)
	resultB := acc.LoadAccount(string(toB))
	assert.Equal(suite.T(), Amount, resultB.Balance)
}

func createAssetTransferTokenTx(s suite.Suite, privFrom string, to []byte) (*types.Transaction, error) {
	param := types.CreateTx{
		To:          string(to),
		Amount:      Amount,
		Fee:         0,
		Note:        "test asset transfer",
		IsWithdraw:  false,
		IsToken:     false,
		TokenSymbol: TestSymbol,
		ExecName:    Title + pt.ParaX,
	}
	tx, err := pt.CreateRawAssetTransferTx(&param)
	assert.Nil(s.T(), err, "create asset transfer failed")
	if err != nil {
		return nil, err
	}

	tx, err = signTx(s, tx, privFrom)
	assert.Nil(s.T(), err, "sign asset transfer failed")
	if err != nil {
		return nil, err
	}

	return tx, nil
}
