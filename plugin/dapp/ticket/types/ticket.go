package types

import (
	"encoding/json"
	"reflect"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const (
	//log for ticket
	TyLogNewTicket   = 111
	TyLogCloseTicket = 112
	TyLogMinerTicket = 113
	TyLogTicketBind  = 114
)

//ticket
const (
	TicketActionGenesis = 11
	TicketActionOpen    = 12
	TicketActionClose   = 13
	TicketActionList    = 14 //读的接口不直接经过transaction
	TicketActionInfos   = 15 //读的接口不直接经过transaction
	TicketActionMiner   = 16
	TicketActionBind    = 17
)

var TicketX = types.TicketX

func init() {
	types.RegistorExecutor(TicketX, NewType())
}

type TicketType struct {
	types.ExecTypeBase
}

func NewType() *TicketType {
	c := &TicketType{}
	c.SetChild(c)
	return c
}

func (at *TicketType) GetPayload() types.Message {
	return &TicketAction{}
}

func (t *TicketType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{
		TyLogNewTicket:   {reflect.TypeOf(ReceiptTicket{}), "TyLogNewTicket"},
		TyLogCloseTicket: {reflect.TypeOf(ReceiptTicket{}), "TyLogCloseTicket"},
		TyLogMinerTicket: {reflect.TypeOf(ReceiptTicket{}), "TyLogMinerTicket"},
		TyLogTicketBind:  {reflect.TypeOf(ReceiptTicketBind{}), "TyLogTicketBind"},
	}
}

func (ticket TicketType) Amount(tx *types.Transaction) (int64, error) {
	var action TicketAction
	err := types.Decode(tx.GetPayload(), &action)
	if err != nil {
		return 0, types.ErrDecode
	}
	if action.Ty == TicketActionMiner && action.GetMiner() != nil {
		ticketMiner := action.GetMiner()
		return ticketMiner.Reward, nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (ticket *TicketType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

func (ticket *TicketType) GetName() string {
	return TicketX
}

func (ticket *TicketType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Genesis": TicketActionGenesis,
		"Topen":   TicketActionOpen,
		"Tbind":   TicketActionBind,
		"Tclose":  TicketActionClose,
		"Miner":   TicketActionMiner,
	}
}
