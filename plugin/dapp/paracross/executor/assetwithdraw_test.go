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

// 构建跨链交易, 依然使用1个节点
// 流程模拟
//   Height 9  withdraw 交易
//   Height 10 有committed, 并在主链执行
//

type AssetWithdrawTestSuite struct {
	suite.Suite
	stateDB dbm.KV
	localDB *dbmock.KVDB
	api     *apimock.QueueProtocolAPI

	exec *Paracross
}

func TestAssetWithdrawSuite(t *testing.T) {
	suite.Run(t, new(AssetWithdrawTestSuite))
}

func (suite *AssetWithdrawTestSuite) SetupTest() {
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

// 主链先不执行
func (suite *AssetWithdrawTestSuite) TestExecAssetWithdrawOnMainChain() {
	types.Init("test", nil)
	tx, err := createAssetWithdrawTx(suite.Suite, PrivKeyA, Nodes[1])
	if err != nil {
		suite.T().Error("createAssetWithdrawTx", "err", err)
		return
	}

	receipt, err := suite.exec.Exec(tx, 1)
	if err != nil {
		suite.T().Error("Exec Transfer", err)
		return
	}
	assert.Nil(suite.T(), receipt, "mainChain not exec withdraw, wait for paraChain")
}

// 平行链执行
func (suite *AssetWithdrawTestSuite) TestExecAssetWithdrawOnParaChain() {
	types.Init(Title, nil)
	// make coins for transfer

	total := 1000 * types.Coin
	accountA := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	paraAcc, _ := NewParaAccount(Title, "coins", "bty", suite.stateDB)
	paraAcc.SaveAccount(&accountA)

	tx, err := createAssetWithdrawTx(suite.Suite, PrivKeyA, Nodes[1])
	if err != nil {
		suite.T().Error("createAssetWithdrawTx", "err", err)
		return
	}

	receipt, err := suite.exec.Exec(tx, 1)
	if err != nil {
		suite.T().Error("Exec Transfer", err)
		return
	}
	assert.Len(suite.T(), receipt.KV, 1, "paraChain exec withdraw")
	for _, kv := range receipt.KV {
		var v types.Account
		err = types.Decode(kv.Value, &v)
		if err != nil {
			// skip, only check frozen
			continue
		}
		suite.T().Log(string(kv.Key), v)
		assert.Equal(suite.T(), total-Amount, v.Balance, "check left coins")
	}
}

// 主链在平行链执行成功后执行
func (suite *AssetWithdrawTestSuite) TestExecAssetWithdrawAfterPara() {
	types.Init("test", nil)
	// make coins for transfer
	acc := account.NewCoinsAccount()
	acc.SetDB(suite.stateDB)

	total := 10 * types.Coin
	pp := 5 * types.Coin
	pb := 5 * types.Coin
	addrPara := address.ExecAddress(Title + pt.ParaX)
	addrMain := address.ExecAddress(pt.ParaX)
	addrB := string(Nodes[1])
	accountPara := types.Account{
		Balance: pp,
		Frozen:  0,
		Addr:    addrPara,
	}
	acc.SaveExecAccount(addrMain, &accountPara)

	accountB := types.Account{
		Balance: pb,
		Frozen:  0,
		Addr:    addrB,
	}
	acc.SaveExecAccount(addrMain, &accountB)

	accountMain := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    addrMain,
	}
	acc.SaveAccount(&accountMain)

	tx, err := createAssetWithdrawTx(suite.Suite, PrivKeyA, Nodes[1])
	if err != nil {
		suite.T().Error("createAssetWithdrawTx", "err", err)
		return
	}

	var payload pt.ParacrossAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		suite.T().Error("decode payload failed", err)
	}
	a := newAction(suite.exec, tx)
	receipt, err := a.assetWithdraw(payload.GetAssetWithdraw(), tx)
	if err != nil {
		suite.T().Error("Exec Transfer", err)
		return
	}
	assert.Len(suite.T(), receipt.KV, 2, "paraChain exec withdraw")

	for _, kv := range receipt.KV {
		var v types.Account
		err = types.Decode(kv.Value, &v)
		if err != nil {
			// skip, only check frozen
			continue
		}
		suite.T().Log(string(kv.Key), v)
		if v.Addr == addrB {
			assert.Equal(suite.T(), pb+Amount, v.Balance, "check to addr coins")
		} else if v.Addr == addrPara {
			assert.Equal(suite.T(), pp-Amount, v.Balance, "check left coins")
		}
	}
}

func (suite *AssetWithdrawTestSuite) TestExecWithdrawFailedOnPara() {
	types.Init(Title, nil)
	// make coins for transfer
	acc := account.NewCoinsAccount()
	acc.SetDB(suite.stateDB)

	addrPara := address.ExecAddress(Title + pt.ParaX)

	total := 1000 * types.Coin
	accountA := types.Account{
		Balance: 0,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	acc.SaveExecAccount(addrPara, &accountA)

	accountExec := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    addrPara,
	}
	acc.SaveAccount(&accountExec)

	tx, err := createAssetWithdrawTx(suite.Suite, PrivKeyA, Nodes[1])
	if err != nil {
		suite.T().Error("createAssetWithdrawTx", "err", err)
		return
	}

	_, err = suite.exec.Exec(tx, 1)
	if err != types.ErrNoBalance {
		suite.T().Error("Exec Transfer", err)
		return
	}
}

func createAssetWithdrawTx(s suite.Suite, privFrom string, to []byte) (*types.Transaction, error) {
	param := types.CreateTx{
		To:          string(to),
		Amount:      Amount,
		Fee:         0,
		Note:        "test asset transfer",
		IsWithdraw:  true,
		IsToken:     false,
		TokenSymbol: "",
		ExecName:    Title + pt.ParaX,
	}
	tx, err := pt.CreateRawAssetTransferTx(&param)
	assert.Nil(s.T(), err, "create asset Withdraw failed")
	if err != nil {
		return nil, err
	}

	tx, err = signTx(s, tx, privFrom)
	assert.Nil(s.T(), err, "sign asset Withdraw failed")
	if err != nil {
		return nil, err
	}

	return tx, err
}
