package paracross

import (
	"encoding/json"

	"math/rand"
	"time"

	log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

// action type
const (
	ParacrossActionCommit = iota
)

// status
const (
	ParacrossStatusCommiting = iota
	ParacrossStatusCommitDone
)

const name = "paracross"

var glog = log.New("module", name)

func Init() {
	// init executor type
	types.RegistorExecutor(name, &ParacrossType{})

	// init log
	types.RegistorLog(types.TyLogParacrossCommit, &ParacrossCommitLog{})
	types.RegistorLog(types.TyLogParacrossDone, &ParacrossDoneLog{})

	// init query rpc
	types.RegistorRpcType("ParacrossGetHeight", &ParacrossGetHeight{})
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
		return "ParacrossCommit"
	}
	return "unkown"
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
	}

	return nil, types.ErrNotSupport
}

func CreateRawParacrossCommitTx(parm *ParacrossCommitTx) (*types.Transaction, error) {
	if parm == nil {
		glog.Error("CreateRawParacrossCommitTx", "parm", parm)
		return nil, types.ErrInvalidParam
	}
	v := &types.ParacrossCommitAction{
		Status: &types.ParacrossNodeStatus{
			StateHash: parm.StateHash,
			Height:    parm.Height,
			Title:     parm.Title,
		},
	}
	action := &types.ParacrossAction{
		Ty:    ParacrossActionCommit,
		Value: &types.ParacrossAction_Commit{v},
	}
	tx := &types.Transaction{
		Execer:  []byte(name),
		Payload: types.Encode(action),
		Fee:     parm.Fee,
		Nonce:   rand.New(rand.NewSource(time.Now().UnixNano())).Int63(),
		To:      address.ExecAddress(name),
	}

	return tx, nil
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

type ParacrossGetHeight struct {
}

func (t *ParacrossGetHeight) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqStr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *ParacrossGetHeight) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
