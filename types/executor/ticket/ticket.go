package ticket

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "ticket"

//var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &TicketType{})

	// init log
	types.RegistorLog(types.TyLogNewTicket, &TicketNewLog{})
	types.RegistorLog(types.TyLogCloseTicket, &TicketCloseLog{})
	types.RegistorLog(types.TyLogMinerTicket, &TicketMinerLog{})
	types.RegistorLog(types.TyLogTicketBind, &TicketBindLog{})

	// init query rpc
	types.RegistorRpcType("TicketInfos", &TicketInfos{})
	types.RegistorRpcType("TicketList", &TicketList{})
	types.RegistorRpcType("MinerAddress", &TicketMinerAddress{})
	types.RegistorRpcType("MinerSourceList", &TicketMinerSourceList{})
}

type TicketType struct {
	types.ExecTypeBase
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
func (ticket TicketType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

type TicketNewLog struct {
}

func (l TicketNewLog) Name() string {
	return "LogNewTicket"
}

func (l TicketNewLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptTicket
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TicketCloseLog struct {
}

func (l TicketCloseLog) Name() string {
	return "LogCloseTicket"
}

func (l TicketCloseLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptTicket
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TicketMinerLog struct {
}

func (l TicketMinerLog) Name() string {
	return "LogMinerTicket"
}

func (l TicketMinerLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptTicket
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type TicketBindLog struct {
}

func (l TicketBindLog) Name() string {
	return "LogTicketBind"
}

func (l TicketBindLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptTicketBind
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

// query
type TicketInfos struct {
}

func (t *TicketInfos) Input(message json.RawMessage) ([]byte, error) {
	var req types.TicketInfos
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TicketInfos) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type TicketList struct {
}

func (t *TicketList) Input(message json.RawMessage) ([]byte, error) {
	var req types.TicketList
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TicketList) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type TicketMinerAddress struct {
}

func (t *TicketMinerAddress) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TicketMinerAddress) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type TicketMinerSourceList struct {
}

func (t *TicketMinerSourceList) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqString
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *TicketMinerSourceList) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
