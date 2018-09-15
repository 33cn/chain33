package paracross

import (
	"encoding/json"

	"math/rand"
	"time"

	"fmt"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

// action type
const (
	ParacrossActionCommit = iota
	ParacrossActionMiner
)
const (
	ParacrossActionTransfer = iota + types.ParaCrossTransferActionTypeStart
	ParacrossActionWithdraw
)

// status
const (
	ParacrossStatusCommiting = iota
	ParacrossStatusCommitDone
)

var (
	ParacrossActionCommitStr   = string("Commit")
	ParacrossActionTransferStr = string("Transfer")
	ParacrossActionWithdrawStr = string("Withdraw")
)

const orgName = "paracross"

var paraVoteHeightKey string
var nameX string

var glog = log.New("module", orgName)

func Init() {
	nameX = types.ExecName(orgName)
	// init executor type
	types.RegistorExecutor(nameX, &ParacrossType{})

	// init log
	types.RegistorLog(types.TyLogParacrossCommit, &ParacrossCommitLog{})
	types.RegistorLog(types.TyLogParacrossCommitDone, &ParacrossDoneLog{})
	types.RegistorLog(types.TyLogParacrossCommitRecord, &ParacrossCommitRecordLog{})
	types.RegistorLog(types.TyLogParaAssetWithdraw, &ParacrossAssetWithdrawLog{})
	types.RegistorLog(types.TyLogParaAssetTransfer, &ParacrossAssetTransferLog{})

	// init query rpc
	types.RegisterRPCQueryHandle("ParacrossGetTitle", &ParacrossGetTitle{})
	types.RegisterRPCQueryHandle("ParacrossListTitles", &ParacrossListTitles{})
	types.RegisterRPCQueryHandle("ParacrossGetTitleHeight", &ParacrossGetTitleHeight{})
	types.RegisterRPCQueryHandle("ParacrossGetAssetTxResult", &ParacrossGetAssetTxResult{})

	paraVoteHeightKey = types.ExecName("paracross") + "-titleVoteHeight-"
}

func CalcMinerHeightKey(title string, height int64) []byte {
	return []byte(fmt.Sprintf(paraVoteHeightKey+"%s-%012d", title, height))
}

func GetExecName() string {
	return nameX
}

type ParacrossType struct {
	types.ExecTypeBase
}

func (m ParacrossType) ActionName(tx *types.Transaction) string {
	var g types.ParacrossAction
	err := types.Decode(tx.Payload, &g)
	if err != nil {
		return "unkown-paracross-action-err"
	}
	if g.Ty == ParacrossActionCommit && g.GetCommit() != nil {
		return ParacrossActionCommitStr
	} else if g.Ty == ParacrossActionTransfer && g.GetAssetTransfer() != nil {
		return ParacrossActionTransferStr
	} else if g.Ty == ParacrossActionWithdraw && g.GetAssetWithdraw() != nil {
		return ParacrossActionWithdrawStr
	} else if g.Ty == ParacrossActionMiner && g.GetMiner() != nil {
		return types.MinerAction
	}
	return "unkown"
}

func (m ParacrossType) DecodePayload(tx *types.Transaction) (interface{}, error) {
	var action types.ParacrossAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return nil, err
	}
	return &action, nil
}

func (m ParacrossType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

func (m ParacrossType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	if action == "ParacrossCommit" {
		var param ParacrossCommitTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}

		return CreateRawParacrossCommitTx(&param)
	} else if action == "ParacrossTransfer" || action == "ParacrossWithdraw" {
		var param types.CreateTx
		err := json.Unmarshal(message, &param)
		if err != nil {
			glog.Error("CreateTx", "Error", err)
			return nil, types.ErrInputPara
		}
		return CreateRawTransferTx(&param)

	}

	return nil, types.ErrNotSupport
}

func CreateRawParacrossCommitTx(parm *ParacrossCommitTx) (*types.Transaction, error) {
	if parm == nil {
		glog.Error("CreateRawParacrossCommitTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}
	return createRawCommitTx(&parm.Status, nameX, parm.Fee)
}

func CreateRawCommitTx4MainChain(status *types.ParacrossNodeStatus, name string, fee int64) (*types.Transaction, error) {
	return createRawCommitTx(status, name, fee)
}

func createRawCommitTx(status *types.ParacrossNodeStatus, name string, fee int64) (*types.Transaction, error) {
	v := &types.ParacrossCommitAction{
		Status: status,
	}
	action := &types.ParacrossAction{
		Ty:    ParacrossActionCommit,
		Value: &types.ParacrossAction_Commit{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(action),
		Fee:     fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func CreateRawMinerTx(status *types.ParacrossNodeStatus) (*types.Transaction, error) {
	v := &types.ParacrossMinerAction{
		Status: status,
	}
	action := &types.ParacrossAction{
		Ty:    ParacrossActionMiner,
		Value: &types.ParacrossAction_Miner{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(nameX),
		Payload: types.Encode(action),
		Nonce:   0, //for consensus purpose, block hash need same, different auth node need keep totally same vote tx
		To:      address.ExecAddress(nameX),
	}

	err := tx.SetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func CreateRawTransferTx(param *types.CreateTx) (*types.Transaction, error) {
	// 跨链交易需要在主链和平行链上执行， 所以应该可以在主链和平行链上构建
	if !types.IsParaExecName(param.GetExecName()) {
		log.Error("CreateRawTransferTx", "exec", param.GetExecName())
		return nil, types.ErrInputPara
	}

	transfer := &types.ParacrossAction{}
	if !param.IsWithdraw {
		v := &types.ParacrossAction_AssetTransfer{AssetTransfer: &types.CoinsTransfer{
			Amount: param.Amount, Note: param.GetNote(), To: param.GetTo()}}
		transfer.Value = v
		transfer.Ty = ParacrossActionTransfer
	} else {
		v := &types.ParacrossAction_AssetWithdraw{AssetWithdraw: &types.CoinsWithdraw{
			Amount: param.Amount, Note: param.GetNote(), To: param.GetTo()}}
		transfer.Value = v
		transfer.Ty = ParacrossActionWithdraw
	}
	tx := &types.Transaction{
		Execer:  []byte(param.GetExecName()),
		Payload: types.Encode(transfer),
		To:      address.ExecAddress(param.GetExecName()),
		Fee:     param.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
	}

	if err := tx.SetRealFee(types.MinFee); err != nil {
		return nil, err
	}

	return tx, nil
}

func CheckMinerTx(current *types.BlockDetail) error {
	//检查第一个笔交易的execs, 以及执行状态
	if len(current.Block.Txs) == 0 {
		return types.ErrEmptyTx
	}
	baseTx := current.Block.Txs[0]
	//判断交易类型和执行情况
	var action types.ParacrossAction
	err := types.Decode(baseTx.GetPayload(), &action)
	if err != nil {
		return err
	}
	if action.GetTy() != ParacrossActionMiner {
		return types.ErrParaMinerTxType
	}
	//判断交易执行是否OK
	if action.GetMiner() == nil {
		return types.ErrParaEmptyMinerTx
	}

	//判断exec 是否成功
	if current.Receipts[0].Ty != types.ExecOk {
		return types.ErrParaMinerExecErr
	}
	return nil
}

type ParacrossCommitLog struct {
}

func (l ParacrossCommitLog) Name() string {
	return "LogParacrossCommit"
}

func (l ParacrossCommitLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptParacrossCommit
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossDoneLog struct {
}

func (l ParacrossDoneLog) Name() string {
	return "LogParacrossDone"
}

func (l ParacrossDoneLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptParacrossDone
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossCommitRecordLog struct {
}

func (l ParacrossCommitRecordLog) Name() string {
	return "LogParacrossCommitRecord"
}

func (l ParacrossCommitRecordLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptParacrossRecord
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossAssetWithdrawLog struct {
}

func (l ParacrossAssetWithdrawLog) Name() string {
	return "LogParacrossAssetWithdraw"
}

func (l ParacrossAssetWithdrawLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossAssetTransferLog struct {
}

func (l ParacrossAssetTransferLog) Name() string {
	return "LogParacrossAssetWithdraw"
}

func (l ParacrossAssetTransferLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptExecAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type ParacrossGetTitle struct {
}

func (t *ParacrossGetTitle) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqStr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *ParacrossGetTitle) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type ParacrossListTitles struct {
}

func (t *ParacrossListTitles) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqNil
	return types.Encode(&req), nil
}

func (t *ParacrossListTitles) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type ParacrossGetTitleHeight struct {
}

func (t *ParacrossGetTitleHeight) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqParacrossTitleHeight
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *ParacrossGetTitleHeight) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}

type ParacrossGetAssetTxResult struct {
}

func (t *ParacrossGetAssetTxResult) JsonToProto(message json.RawMessage) ([]byte, error) {
	var req types.ReqHash
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *ParacrossGetAssetTxResult) ProtoToJson(reply *types.Message) (interface{}, error) {
	return reply, nil
}
