package types

import (
	"errors"
)

//log type
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

//hard fork block height
const (
	ForkV7AddRelay = 1
)

const (
	RelayOrderCreate = iota
	RelayOrderAccept
)

//relay action ty
const (
	RelayActionCreate = iota
	RelayActionAccept
	RelayActionRevoke
	RelayActionConfirmTx
	RelayActionVerifyTx
	RelayActionVerifyBTCTx
	RelayActionRcvBTCHeaders
)

const (
	RelayCreateBuy = iota
	RelayCreateSell
)

const (
	RelayUnlockOrder = iota
	RelayCancleOrder
)

/////////////////error.go/////
//err for relay
var (
	ErrTRelayBalanceNotEnough   = errors.New("ErrRelaySellBalanceNotEnough")
	ErrTRelayOrderNotExist      = errors.New("ErrRelayOrderNotExist")
	ErrTRelayOrderOnSell        = errors.New("ErrRelayOrderOnSell")
	ErrTRelayOrderParamErr      = errors.New("ErrRelayOrderParamErr")
	ErrTRelayOrderSoldout       = errors.New("ErrRelayOrderSoldout")
	ErrTRelayOrderRevoked       = errors.New("ErrRelayOrderRevoked")
	ErrTRelayOrderConfirming    = errors.New("ErrRelayOrderConfirming")
	ErrTRelayOrderFinished      = errors.New("ErrRelayOrderFinished")
	ErrTRelayReturnAddr         = errors.New("ErrRelayReturnAddr")
	ErrTRelayVerify             = errors.New("ErrRelayVerify")
	ErrTRelayVerifyAddrNotFound = errors.New("ErrRelayVerifyAddrNotFound")
	ErrTRelayWaitBlocksErr      = errors.New("ErrRelayWaitBlocks")
)
