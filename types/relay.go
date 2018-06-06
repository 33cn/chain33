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
	ErrRelayBalanceNotEnough   = errors.New("ErrRelaySellBalanceNotEnough")
	ErrRelayOrderNotExist      = errors.New("ErrRelayOrderNotExist")
	ErrRelayOrderOnSell        = errors.New("ErrRelayOrderOnSell")
	ErrRelayOrderParamErr      = errors.New("ErrRelayOrderParamErr")
	ErrRelayOrderSoldout       = errors.New("ErrRelayOrderSoldout")
	ErrRelayOrderRevoked       = errors.New("ErrRelayOrderRevoked")
	ErrRelayOrderConfirming    = errors.New("ErrRelayOrderConfirming")
	ErrRelayOrderFinished      = errors.New("ErrRelayOrderFinished")
	ErrRelayReturnAddr         = errors.New("ErrRelayReturnAddr")
	ErrRelayVerify             = errors.New("ErrRelayVerify")
	ErrRelayVerifyAddrNotFound = errors.New("ErrRelayVerifyAddrNotFound")
	ErrRelayWaitBlocksErr      = errors.New("ErrRelayWaitBlocks")
)
