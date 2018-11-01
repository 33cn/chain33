package types

import (
	"encoding/json"
	"reflect"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var RelayX = "relay"

//var tlog = log.New("module", name)
//log for relay
const (
	TyLogRelayCreate       = 350
	TyLogRelayRevokeCreate = 351
	TyLogRelayAccept       = 352
	TyLogRelayRevokeAccept = 353
	TyLogRelayConfirmTx    = 354
	TyLogRelayFinishTx     = 355
	TyLogRelayRcvBTCHead   = 356
)

// relay
const (
	RelayRevokeCreate = iota
	RelayRevokeAccept
)

const (
	RelayOrderBuy = iota
	RelayOrderSell
)

var RelayOrderOperation = map[uint32]string{
	RelayOrderBuy:  "buy",
	RelayOrderSell: "sell",
}

const (
	RelayUnlock = iota
	RelayCancel
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(RelayX))
	types.RegistorExecutor(RelayX, NewType())
	types.RegisterDappFork(RelayX, "Enable", 570000)
}

func NewType() *RelayType {
	c := &RelayType{}
	c.SetChild(c)
	return c
}

func (at *RelayType) GetPayload() types.Message {
	return &RelayAction{}
}

type RelayType struct {
	types.ExecTypeBase
}

func (b *RelayType) GetName() string {
	return RelayX
}

func (t *RelayType) GetLogMap() map[int64]*types.LogInfo {
	return map[int64]*types.LogInfo{
		TyLogRelayCreate:       {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayCreate"},
		TyLogRelayRevokeCreate: {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayRevokeCreate"},
		TyLogRelayAccept:       {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayAccept"},
		TyLogRelayRevokeAccept: {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayRevokeAccept"},
		TyLogRelayConfirmTx:    {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayConfirmTx"},
		TyLogRelayFinishTx:     {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayFinishTx"},
		TyLogRelayRcvBTCHead:   {reflect.TypeOf(ReceiptRelayRcvBTCHeaders{}), "LogRelayRcvBTCHead"},
	}
}

const (
	RelayActionCreate = iota
	RelayActionAccept
	RelayActionRevoke
	RelayActionConfirmTx
	RelayActionVerifyTx
	RelayActionVerifyCmdTx
	RelayActionRcvBTCHeaders
)

func (t *RelayType) GetTypeMap() map[string]int32 {
	return map[string]int32{
		"Create":     RelayActionCreate,
		"Accept":     RelayActionAccept,
		"Revoke":     RelayActionRevoke,
		"ConfirmTx":  RelayActionConfirmTx,
		"Verify":     RelayActionVerifyTx,
		"VerifyCli":  RelayActionVerifyCmdTx,
		"BtcHeaders": RelayActionRcvBTCHeaders,
	}
}

func (r RelayType) ActionName(tx *types.Transaction) string {
	var relay RelayAction
	err := types.Decode(tx.Payload, &relay)
	if err != nil {
		return "unknown-relay-action-err"
	}
	if relay.Ty == RelayActionCreate && relay.GetCreate() != nil {
		return "relayCreateTx"
	}
	if relay.Ty == RelayActionRevoke && relay.GetRevoke() != nil {
		return "relayRevokeTx"
	}
	if relay.Ty == RelayActionAccept && relay.GetAccept() != nil {
		return "relayAcceptTx"
	}
	if relay.Ty == RelayActionConfirmTx && relay.GetConfirmTx() != nil {
		return "relayConfirmTx"
	}
	if relay.Ty == RelayActionVerifyTx && relay.GetVerify() != nil {
		return "relayVerifyTx"
	}
	if relay.Ty == RelayActionRcvBTCHeaders && relay.GetBtcHeaders() != nil {
		return "relay-receive-btc-heads"
	}
	return "unknown"
}

func (r *RelayType) Amount(tx *types.Transaction) (int64, error) {
	data, err := r.DecodePayload(tx)
	if err != nil {
		return 0, err
	}
	relay := data.(*RelayAction)
	if RelayActionCreate == relay.Ty && relay.GetCreate() != nil {
		return int64(relay.GetCreate().BtyAmount), nil
	}
	return 0, nil
}

// TODO 暂时不修改实现， 先完成结构的重构
func (r *RelayType) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	return tx, nil
}

type RelayCreateLog struct {
}

func (l RelayCreateLog) Name() string {
	return "LogRelayCreate"
}

func (l RelayCreateLog) Decode(msg []byte) (interface{}, error) {
	var logTmp ReceiptRelayLog
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
	var logTmp ReceiptRelayLog
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
	var logTmp ReceiptRelayLog
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
	var logTmp ReceiptRelayLog
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
	var logTmp ReceiptRelayLog
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
	var logTmp ReceiptRelayLog
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
	var logTmp ReceiptRelayRcvBTCHeaders
	err := types.Decode(msg, &logTmp)
	if err != nil {
		return nil, err
	}
	return logTmp, err
}
