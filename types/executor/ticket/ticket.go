package ticket


import (
	"gitlab.33.cn/chain33/chain33/types"
	"encoding/json"
	"time"
	log "github.com/inconshreveable/log15"
	"math/rand"
)

const name = "ticket"

var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &TicketType{})

	// init log
	types.RegistorLog(types.TyLogDeposit, &CoinsDepositLog{})

	// init query rpc
	types.RegistorRpcType("q2", &CoinsGetTxsByAddr{})
}



type TicketType struct {
}

func (ticket TicketType) ActionName(tx *types.Transaction) string {
	var action types.TicketAction
	err := types.Decode(tx.Payload, &action)
	if err != nil {
		return "unknow-err"
	}
	if action.Ty == types.TicketActionGenesis && action.GetGenesis() != nil {
		return "genesis"
	} else if action.Ty == types.TicketActionOpen && action.GetTopen() != nil {
		return "open"
	} else if action.Ty == types.TicketActionClose && action.GetTclose() != nil {
		return "close"
	} else if action.Ty == types.TicketActionMiner && action.GetMiner() != nil {
		return "miner"
	} else if action.Ty == types.TicketActionBind && action.GetTbind() != nil {
		return "bindminer"
	}
	return "unknow"
}

func (ticket TicketType) Amount(tx *types.Transaction) (int64, error) {
	var action types.TicketAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		return 0, types.ErrDecode
	}
	if action.Ty == types.TicketActionMiner && action.GetMiner() != nil {
		ticketMiner := action.GetMiner()
		return ticketMiner.Reward, nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (ticket TicketType) NewTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

type CoinsDepositLog struct {
}

func (l CoinsDepositLog) Name() string {
	return "LogDeposit"
}

func (l CoinsDepositLog) Decode(msg []byte) (interface{}, error){
	var logTmp types.ReceiptAccountTransfer
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type CoinsGetTxsByAddr struct {
}

func (t *CoinsGetTxsByAddr) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqAddr
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *CoinsGetTxsByAddr) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}


