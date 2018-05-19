package types

import (
	"errors"
)

//log type
const (

	//log for relay
	TyLogRelaySell       = 350
	TyLogRelayRevokeSell = 351
	TyLogRelayBuy        = 352
	TyLogRelayRevokeBuy  = 353
	TyLogRelayRcvBTCHead = 354
	TyLogRelayConfirmTx  = 355
)

//hard fork block height
const (
	ForkV7AddRelay = 1
)

//relay action ty
const (
	RelayActionSell = iota
	RelayActionRevokeSell
	RelayActionBuy
	RelayActionRevokeBuy
	RelayActionConfirmTx
	RelayActionVerifyTx
	RelayActionVerifyBTCTx
	RelayActionRcvBTCHeaders
)

/////////////////error.go/////
//err for relay
var (
	ErrTRelayBalanceNotEnough = errors.New("ErrRelaySellBalanceNotEnough")
	ErrTRelayOrderNotExist    = errors.New("ErrRelayOrderNotExist")
	ErrTRelayOrderOnSell      = errors.New("ErrRelayOrderOnSell")
	ErrTRelayOrderSoldout     = errors.New("ErrRelayOrderSoldout")
	ErrTRelayOrderRevoked     = errors.New("ErrRelayOrderRevoked")
	ErrTRelayOrderConfirming  = errors.New("ErrRelayOrderConfirming")
	ErrTRelayOrderFinished    = errors.New("ErrRelayOrderFinished")
	ErrTRelayReturnAddr       = errors.New("ErrRelayReturnAddr")
	ErrTRelayVerify           = errors.New("ErrRelayVerify")
	ErrTRelayVrfAddrNotFound  = errors.New("ErrRelayVerifyAddrNotFound")
)
