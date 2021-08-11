package executor

import (
	"errors"

	"github.com/33cn/chain33/common"
	"github.com/33cn/chain33/system/dapp"
	nty "github.com/33cn/chain33/system/dapp/none/types"
	"github.com/33cn/chain33/types"
)

var (
	errNilDelayTx        = errors.New("errNilDelayTx")
	errDuplicateDelayTx  = errors.New("errDuplicateDelayTx")
	errNegativeDelayTime = errors.New("errNegativeDelayTime")
	errDecodeDelayTx     = errors.New("errDecodeDelayTx")
)

// CheckTx 实现自定义检验交易接口，供框架调用
func (n *None) CheckTx(tx *types.Transaction, index int) error {

	// 通用判定，none交易的to地址必须为执行器地址
	if dapp.ExecAddress(string(tx.Execer)) != tx.To {
		return types.ErrToAddrNotSameToExecAddr
	}

	action := &nty.NoneAction{}
	err := types.Decode(tx.Payload, action)
	// 存证交易不需要执行，没有具体的交易类型，直接返回
	if err != nil {
		return nil
	}

	// 根据定义的交易类型进行相关判定
	if action.Ty == nty.TyCommitDelayTxAction {
		err = n.checkCommitDelayTx(tx, action.GetCommitDelayTx(), index)
	}

	if err != nil {
		eLog.Error("none CheckTx", "txHash", common.ToHex(tx.Hash()), "actionTy", action.Ty)
	}

	return err

}

func (n *None) checkCommitDelayTx(tx *types.Transaction, commit *nty.CommitDelayTx, index int) error {

	delayTx := &types.Transaction{}
	txByte, err := common.FromHex(commit.GetDelayTx())
	if err != nil || types.Decode(txByte, delayTx) != nil {
		return errDecodeDelayTx
	}
	if delayTx.GetSignature() == nil {
		return errNilDelayTx
	}

	if commit.GetRelativeDelayTime() < 0 {
		return errNegativeDelayTime
	}

	_, err = n.GetStateDB().Get(formatDelayTxKey(delayTx.Hash()))
	if err == nil {
		return errDuplicateDelayTx
	}

	return nil
}
