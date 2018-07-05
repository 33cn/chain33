package relay

import (
	"encoding/json"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

const name = "relay"

//var tlog = log.New("module", name)

func init() {
	// init executor type
	types.RegistorExecutor(name, &RelayType{})

	// init log
	types.RegistorLog(types.TyLogRelayCreate, &RelayCreateLog{})
	types.RegistorLog(types.TyLogRelayRevokeCreate, &RelayRevokeCreateLog{})
	types.RegistorLog(types.TyLogRelayAccept, &RelayAcceptLog{})
	types.RegistorLog(types.TyLogRelayRevokeAccept, &RelayRevokeAcceptLog{})
	types.RegistorLog(types.TyLogRelayConfirmTx, &RelayConfirmTxLog{})
	types.RegistorLog(types.TyLogRelayFinishTx, &RelayFinishTxLog{})
	types.RegistorLog(types.TyLogRelayRcvBTCHead, &RelayRcvBTCHeadLog{})

	// init query rpc
	types.RegistorRpcType("GetRelayOrderByStatus", &RelayGetRelayOrderByStatus{})
	types.RegistorRpcType("GetSellRelayOrder", &RelayGetSellRelayOrder{})
	types.RegistorRpcType("GetBuyRelayOrder", &RelayGetBuyRelayOrder{})
	types.RegistorRpcType("GetBTCHeaderList", &RelayGetBTCHeaderList{})
	types.RegistorRpcType("GetBTCHeaderMissList", &RelayGetBTCHeaderMissList{})
	types.RegistorRpcType("GetBTCHeaderCurHeight", &RelayGetBTCHeaderCurHeight{})

}

type RelayType struct {
}

func (r RelayType) ActionName(tx *types.Transaction) string {
	var relay types.RelayAction
	err := types.Decode(tx.Payload, &relay)
	if err != nil {
		return "unkown-relay-action-err"
	}
	if relay.Ty == types.RelayActionCreate && relay.GetCreate() != nil {
		return "relayCreateTx"
	}
	if relay.Ty == types.RelayActionRevoke && relay.GetRevoke() != nil {
		return "relayRevokeTx"
	}
	if relay.Ty == types.RelayActionAccept && relay.GetAccept() != nil {
		return "relayAcceptTx"
	}
	if relay.Ty == types.RelayActionConfirmTx && relay.GetConfirmTx() != nil {
		return "relayConfirmTx"
	}
	if relay.Ty == types.RelayActionVerifyTx && relay.GetVerify() != nil {
		return "relayVerifyTx"
	}
	if relay.Ty == types.RelayActionRcvBTCHeaders && relay.GetBtcHeaders() != nil {
		return "relay-receive-btc-heads"
	}
	return "unknow"
}

func (r RelayType) Amount(tx *types.Transaction) (int64, error) {
	var relay types.RelayAction
	err := types.Decode(tx.GetPayload(), &relay)
	if err != nil {
		return 0, types.ErrDecode
	}
	if types.RelayActionCreate == relay.Ty && relay.GetCreate() != nil {
		return int64(relay.GetCreate().BtyAmount), nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (r RelayType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

type RelayCreateLog struct {
}

func (l RelayCreateLog) Name() string {
	return "LogRelayCreate"
}

func (l RelayCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptRelayLog
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RelayRevokeCreateLog struct {
}

func (l RelayRevokeCreateLog) Name() string {
	return "LogRelayRevokeCreate"
}

func (l RelayRevokeCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptRelayLog
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RelayAcceptLog struct {
}

func (l RelayAcceptLog) Name() string {
	return "LogRelayAccept"
}

func (l RelayAcceptLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptRelayLog
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RelayRevokeAcceptLog struct {
}

func (l RelayRevokeAcceptLog) Name() string {
	return "LogRelayRevokeAccept"
}

func (l RelayRevokeAcceptLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptRelayLog
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RelayConfirmTxLog struct {
}

func (l RelayConfirmTxLog) Name() string {
	return "LogRelayConfirmTx"
}

func (l RelayConfirmTxLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptRelayLog
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RelayFinishTxLog struct {
}

func (l RelayFinishTxLog) Name() string {
	return "LogRelayFinishTx"
}

func (l RelayFinishTxLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptRelayLog
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RelayRcvBTCHeadLog struct {
}

func (l RelayRcvBTCHeadLog) Name() string {
	return "LogRelayRcvBTCHead"
}

func (l RelayRcvBTCHeadLog) Decode(msg []byte) (interface{}, error) {
	var logTmp types.ReceiptRelayRcvBTCHeaders
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}

type RelayGetRelayOrderByStatus struct {
}

func (t *RelayGetRelayOrderByStatus) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRelayAddrCoins
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RelayGetRelayOrderByStatus) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type RelayGetSellRelayOrder struct {
}

func (t *RelayGetSellRelayOrder) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRelayAddrCoins
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RelayGetSellRelayOrder) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type RelayGetBuyRelayOrder struct {
}

func (t *RelayGetBuyRelayOrder) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRelayAddrCoins
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RelayGetBuyRelayOrder) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type RelayGetBTCHeaderList struct {
}

func (t *RelayGetBTCHeaderList) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRelayBtcHeaderHeightList
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RelayGetBTCHeaderList) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type RelayGetBTCHeaderMissList struct {
}

func (t *RelayGetBTCHeaderMissList) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRelayBtcHeaderHeightList
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RelayGetBTCHeaderMissList) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}

type RelayGetBTCHeaderCurHeight struct {
}

func (t *RelayGetBTCHeaderCurHeight) Input(message json.RawMessage) ([]byte, error) {
	var req types.ReqRelayQryBTCHeadHeight
	err := json.Unmarshal(message, &req)
	if err != nil {
		return nil, err
	}
	return types.Encode(&req), nil
}

func (t *RelayGetBTCHeaderCurHeight) Output(reply interface{}) (interface{}, error) {
	return reply, nil
}
