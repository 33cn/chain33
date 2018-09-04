package paracross

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
	"gitlab.33.cn/chain33/chain33/types"
	pt "gitlab.33.cn/chain33/chain33/types/executor/paracross"
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
	var titleStatus types.ParacrossStatus
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
	types.SetTitle("test")
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
	types.SetTitle(Title)
	// make coins for transfer
	acc := account.NewCoinsAccount()
	acc.SetDB(suite.stateDB)

	total := 1000 * types.Coin
	accountA := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	acc.SaveExecAccount(suite.exec.GetAddr(), &accountA)

	accountExec := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    suite.exec.GetAddr(),
	}
	acc.SaveAccount(&accountExec)

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
	assert.Len(suite.T(), receipt.KV, 2, "paraChain exec withdraw")
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
	types.SetTitle("test")
	// make coins for transfer
	acc := account.NewCoinsAccount()
	acc.SetDB(suite.stateDB)

	total := 10 * types.Coin
	pp := 5 * types.Coin
	pb := 5 * types.Coin
	addrPara := address.ExecAddress(Title + types.ParaX)
	addrMain := address.ExecAddress(types.ParaX)
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

	var payload types.ParacrossAction
	err = types.Decode(tx.Payload, &payload)
	if err != nil {
		suite.T().Error("decode payload failed", err)
	}
	a := newAction(suite.exec, tx)
	receipt, err := a.assetWithdrawCoins(payload.GetAssetWithdraw(), tx)
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
	types.SetTitle(Title)
	// make coins for transfer
	acc := account.NewCoinsAccount()
	acc.SetDB(suite.stateDB)

	total := 1000 * types.Coin
	accountA := types.Account{
		Balance: 0,
		Frozen:  0,
		Addr:    string(Nodes[0]),
	}
	acc.SaveExecAccount(suite.exec.GetAddr(), &accountA)

	accountExec := types.Account{
		Balance: total,
		Frozen:  0,
		Addr:    suite.exec.GetAddr(),
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
		ExecName:    Title + types.ParaX,
	}
	tx, err := pt.CreateRawTransferTx(&param)
	assert.Nil(s.T(), err, "create asset Withdraw failed")
	if err != nil {
		return tx, err
	}

	tx, err = signTx(s, tx, privFrom)
	assert.Nil(s.T(), err, "sign asset Withdraw failed")
	if err != nil {
		return tx, err
	}

	return tx, err
}

/*
func checkDoneReceiptWithTxMap(suite suite.Suite, receipt *types.Receipt, commitCnt int, paraTxResult bool) {
	assert.Equal(suite.T(), receipt.Ty, int32(types.ExecOk))
	kvLen := 4
	if !paraTxResult {
		kvLen = 3
	}
	assert.Len(suite.T(), receipt.KV, kvLen)

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
	suite.T().Log("title key", string(keyTitle), "receipt key", len(receipt.KV))
	assert.Equal(suite.T(), keyTitle, receipt.KV[1].Key,
		"receipt not match", string(keyTitle), string(receipt.KV[1].Key))

	var titleStat types.ParacrossStatus
	err = types.Decode(receipt.KV[1].Value, &titleStat)
	assert.Nil(suite.T(), err, "decode title failed")
	suite.T().Log("title", titleStat)
	assert.Equal(suite.T(), int32(types.TyLogParacrossCommitDone), receipt.Logs[1].Ty)
	assert.Equal(suite.T(), int64(TitleHeight), titleStat.Height)
	assert.Equal(suite.T(), Title, titleStat.Title)
	assert.Equal(suite.T(), CurBlock, titleStat.BlockHash)

	if paraTxResult == true {
		var transferFrom types.Account
		err = types.Decode(receipt.KV[2].Value, &transferFrom)
		assert.Nil(suite.T(), err, "decode transferFrom")
		suite.T().Log(transferFrom)
		assert.Equal(suite.T(), int64(0), transferFrom.Frozen)
		assert.Equal(suite.T(), string(Nodes[0]), transferFrom.Addr)

		var transferTo types.Account
		err = types.Decode(receipt.KV[3].Value, &transferTo)
		assert.Nil(suite.T(), err, "decode transferTo")
		suite.T().Log(transferTo)
		assert.Equal(suite.T(), Amount, transferTo.Balance)
		assert.Equal(suite.T(), string(Nodes[1]), transferTo.Addr)
	} else {
		var transferFrom types.Account
		err = types.Decode(receipt.KV[2].Value, &transferFrom)
		assert.Nil(suite.T(), err, "decode transferFrom")
		suite.T().Log(transferFrom)
		assert.Equal(suite.T(), int64(0), transferFrom.Frozen)
		assert.Equal(suite.T(), string(Nodes[0]), transferFrom.Addr)
	}
}

func commitOnceImplWithTxMap(suite suite.Suite, exec *Paracross, privkeyStr string, paraTxResult bool) (receipt *types.Receipt) {
	tx, err := fillRawCommitTxWithTxMap(suite, uint32(1), paraTxResult)
	tx, err = signTx(suite, tx, privkeyStr)

	suite.T().Log(tx.From())
	receipt, err = exec.Exec(tx, 0)
	suite.T().Log(receipt)
	assert.NotNil(suite.T(), receipt)
	assert.Nil(suite.T(), err)

	return
}

func fillRawCommitTxWithTxMap(suite suite.Suite, cnt uint32, execResult bool) (*types.Transaction, error) {
	x := byte(0x00)
	if execResult {
		x = byte(0xff)
	}
	st1 := types.ParacrossNodeStatus{
		MainBlockHash10,
		MainBlockHeight,
		Title,
		TitleHeight,
		[]byte("block-hash-9"),
		[]byte("block-hash-10"),
		[]byte("state-hash-9"),
		[]byte("state-hash-10"),
		cnt,
		[]byte{x},
	}
	tx, err := pt.CreateRawCommitTx4MainChain(&st1, types.ParaX, 0)
	if err != nil {
		suite.T().Error("TestExec", "create tx failed", err)
	}
	return tx, err
}

func mockExecParaTx(tx *types.Transaction, suite *AssetWithdrawTestSuite) error {
	// 第一个跨链交易时， 没有记录
	cntKey := calcLocalTxsCntKey([]byte(Title))
	suite.localDB.On("Get", cntKey).Return(nil, types.ErrNotFound)

	// 平行链交易在本地信息
	txResult := types.TxResult{
		Height: MainBlockHeightForTransfer,
		Tx:     tx,
	}
	suite.T().Log(txResult)
	suite.localDB.On("Get", tx.Hash()).Return(types.Encode(&txResult), nil)

	//

	return nil
}

*/
