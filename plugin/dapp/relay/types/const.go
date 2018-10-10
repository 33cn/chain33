package types

import (
	"reflect"

	"github.com/inconshreveable/log15"
	"gitlab.33.cn/chain33/chain33/types"
)

//relay action ty
const (
	RelayActionCreate = iota
	RelayActionAccept
	RelayActionRevoke
	RelayActionConfirmTx
	RelayActionVerifyTx
	RelayActionVerifyCmdTx
	RelayActionRcvBTCHeaders
)

const (
	//log for relay
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

const (
	GetBlackwhiteRoundInfo       = "GetBlackwhiteRoundInfo"
	GetBlackwhiteByStatusAndAddr = "GetBlackwhiteByStatusAndAddr"
	GetBlackwhiteloopResult      = "GetBlackwhiteloopResult"
)

var (
	RelayX      = "relay"
	glog        = log15.New("module", RelayX)
	JRPCName    = "Relay"
	ExecerRelay = []byte(RelayX)
	actionName  = map[string]int32{
		"Create":     RelayActionCreate,
		"Accept":     RelayActionAccept,
		"Revoke":     RelayActionRevoke,
		"ConfirmTx":  RelayActionConfirmTx,
		"Verify":     RelayActionVerifyTx,
		"verifyCli":  RelayActionVerifyCmdTx,
		"BtcHeaders": RelayActionRcvBTCHeaders,
	}
	logInfo = map[int64]*types.LogInfo{
		TyLogRelayCreate:       {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayCreate"},
		TyLogRelayRevokeCreate: {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayRevokeCreate"},
		TyLogRelayAccept:       {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayAccept"},
		TyLogRelayRevokeAccept: {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayRevokeAccept"},
		TyLogRelayConfirmTx:    {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayConfirmTx"},
		TyLogRelayFinishTx:     {reflect.TypeOf(ReceiptRelayLog{}), "LogRelayFinishTx"},
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, ExecerRelay)
}
