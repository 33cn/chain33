package executor

import (
	"gitlab.33.cn/chain33/chain33/types"
	_ "gitlab.33.cn/chain33/chain33/types/executor/coins"
	_ "gitlab.33.cn/chain33/chain33/types/executor/evm"
	_ "gitlab.33.cn/chain33/chain33/types/executor/hashlock"
	_ "gitlab.33.cn/chain33/chain33/types/executor/none"
	_ "gitlab.33.cn/chain33/chain33/types/executor/retrieve"
	_ "gitlab.33.cn/chain33/chain33/types/executor/ticket"
	_ "gitlab.33.cn/chain33/chain33/types/executor/token"
	_ "gitlab.33.cn/chain33/chain33/types/executor/trade"
)

// 进度：
// coins: 		actionName	NewTx	log		query
// evm: 		actionName
// game:
// hashlock: 	actionName
// manage:
// none: 		actionName
// retrieve: 	actionName
// ticket:		actionName
// token:		actionName	NewTx	log		query
// trade:		actionName	NewTx	log		query

func init() {
	// not need to init executor

	// init common log
	types.RegistorLog(types.TyLogErr, &ErrLog{})
	types.RegistorLog(types.TyLogFee, &FeeLog{})

	// init query rpc type
}

type ErrLog struct {
}

func (l ErrLog) Name() string {
	return "LogErr"
}

func (l ErrLog) Decode(msg []byte) (interface{}, error){
	return string(msg), nil
}

type FeeLog struct {
}

func (l FeeLog) Name() string {
	return "LogFee"
}

func (l FeeLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}
