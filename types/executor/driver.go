package executor

import (
	"sync"

	"gitlab.33.cn/chain33/chain33/types"
	"gitlab.33.cn/chain33/chain33/types/executor/evm"
	"gitlab.33.cn/chain33/chain33/types/executor/manage"
	"gitlab.33.cn/chain33/chain33/types/executor/relay"
	"gitlab.33.cn/chain33/chain33/types/executor/retrieve"
	"gitlab.33.cn/chain33/chain33/types/executor/ticket"
	"gitlab.33.cn/chain33/chain33/types/executor/token"
	"gitlab.33.cn/chain33/chain33/types/executor/trade"
)

// 进度：
// 	ActionName  done
//	Amount 		done
//	Log			done
// coins: 		actionName	CreateTx	log		query	Amount
// evm: 		actionName			Log		query
// game:
// hashlock: 	actionName
// manage:		actionName 			log		query		Amount
// none: 		actionName
// retrieve: 	actionName					query
// ticket:		actionName			log		query		Amount
// token:		actionName	CreateTx	log		query	Amount
// trade:		actionName	CreateTx	log		query	Amount

var once sync.Once

func Init() {
	once.Do(initExec)
}

func initExec() {

	// init common log
	types.RegistorLog(types.TyLogErr, &ErrLog{})
	types.RegistorLog(types.TyLogFee, &FeeLog{})

	// init query rpc type

	//avoid init for ExecPrifex
	evm.Init()
	manage.Init()
	relay.Init()
	retrieve.Init()
	ticket.Init()
	token.Init()
	trade.Init()
}

type ErrLog struct {
}

func (l ErrLog) Name() string {
	return "LogErr"
}

func (l ErrLog) Decode(msg []byte) (interface{}, error) {
	return string(msg), nil
}

type FeeLog struct {
}

func (l FeeLog) Name() string {
	return "LogFee"
}

func (l FeeLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}
