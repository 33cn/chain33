package types

import (
	"encoding/json"
	"reflect"

	//log "github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

var RelayX = types.RelayX

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
