package types

import (
	"encoding/json"
	"log"
	"math/rand"
	"reflect"
	"time"

	"gitlab.33.cn/chain33/chain33/common/address"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	CoinsX                    = "coins"
	CoinsActionTransfer       = 1
	CoinsActionGenesis        = 2
	CoinsActionWithdraw       = 3
	CoinsActionTransferToExec = 10
)

var (
	ExecerCoins = []byte(CoinsX)
	nameX       string
	tlog        = log.New("module", "exectype.coins")

	/*
		对应 proto type 的字段
		//	*CoinsAction_Transfer
		//	*CoinsAction_Withdraw
		//	*CoinsAction_Genesis
		//	*CoinsAction_TransferToExec
	*/
	actionName = map[string]int32{
		"Transfer":       CoinsActionTransfer,
		"TransferToExec": CoinsActionTransferToExec,
		"Withdraw":       CoinsActionWithdraw,
		"Genesis":        CoinsActionGenesis,
	}
	actionFunList = make(map[string]reflect.Method)
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerCoins)
	actionFunList = drivers.ListMethod(&CoinsAction{})
}

func Init() {
	nameX = types.ExecName("coins")
	// init executor type
	types.RegistorExecutor("coins", NewType())
}

type CoinsType struct {
	types.ExecTypeBase
}

func NewType() *CoinsType {
	c := &CoinsType{}
	c.SetChild(c)
	return c
}

func (coins *CoinsType) GetPayload() types.Message {
	return &cty.CoinsAction{}
}

func (coins *CoinsType) GetLogMap() map[int64]reflect.Type {
	return nil
}

func (c *CoinsType) GetTypeMap() map[string]int32 {
	return actionName
}

func (c *CoinsType) GetFuncMap() map[string]reflect.Method {
	return actionFunList
}

// TODO 暂时不修改实现， 先完成结构的重构
func (coins *CoinsType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var param types.CreateTx
	err := json.Unmarshal(message, &param)
	if err != nil {
		tlog.Error("CreateTx", "Error", err)
		return nil, types.ErrInputPara
	}
	if param.ExecName != "" && !types.IsAllowExecName([]byte(param.ExecName), []byte(param.ExecName)) {
		tlog.Error("CreateTx", "Error", types.ErrExecNameNotMatch)
		return nil, types.ErrExecNameNotMatch
	}

	//to地址要么是普通用户地址，要么就是执行器地址，不能为空
	if param.To == "" {
		return nil, types.ErrAddrNotExist
	}

	var tx *types.Transaction
	if param.Amount < 0 {
		return nil, types.ErrAmount
	}
	if param.IsToken {
		return nil, types.ErrNotSupport
	} else {
		tx = CreateCoinsTransfer(&param)
	}

	tx.Fee, err = tx.GetRealFee(types.MinFee)
	if err != nil {
		return nil, err
	}

	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	tx.Nonce = random.Int63()

	return tx, nil
}

func CreateCoinsTransfer(param *types.CreateTx) *types.Transaction {
	transfer := &cty.CoinsAction{}
	to := ""
	if types.IsPara() {
		to = param.GetTo()
	}
	if !param.IsWithdraw {
		if param.ExecName != "" {
			v := &cty.CoinsAction_TransferToExec{TransferToExec: &types.AssetsTransferToExec{
				Amount: param.Amount, Note: param.GetNote(), ExecName: param.GetExecName(), To: to}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionTransferToExec
		} else {
			v := &cty.CoinsAction_Transfer{Transfer: &types.AssetsTransfer{
				Amount: param.Amount, Note: param.GetNote(), To: to}}
			transfer.Value = v
			transfer.Ty = cty.CoinsActionTransfer
		}
	} else {
		v := &cty.CoinsAction_Withdraw{Withdraw: &types.AssetsWithdraw{
			Amount: param.Amount, Note: param.GetNote(), ExecName: param.GetExecName(), To: to}}
		transfer.Value = v
		transfer.Ty = cty.CoinsActionWithdraw
	}
	if types.IsPara() {
		return &types.Transaction{Execer: []byte(nameX), Payload: types.Encode(transfer), To: address.ExecAddress(nameX)}
	}
	return &types.Transaction{Execer: []byte(nameX), Payload: types.Encode(transfer), To: param.GetTo()}
}
