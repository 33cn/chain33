package types

import "errors"

var (
	ErrRelayBalanceNotEnough   = errors.New("ErrRelaySellBalanceNotEnough")
	ErrRelayOrderNotExist      = errors.New("ErrRelayOrderNotExist")
	ErrRelayOrderOnSell        = errors.New("ErrRelayOrderOnSell")
	ErrRelayOrderStatusErr     = errors.New("ErrRelayOrderStatusErr")
	ErrRelayOrderParamErr      = errors.New("ErrRelayOrderParamErr")
	ErrRelayOrderSoldout       = errors.New("ErrRelayOrderSoldout")
	ErrRelayOrderRevoked       = errors.New("ErrRelayOrderRevoked")
	ErrRelayOrderConfirming    = errors.New("ErrRelayOrderConfirming")
	ErrRelayOrderFinished      = errors.New("ErrRelayOrderFinished")
	ErrRelayReturnAddr         = errors.New("ErrRelayReturnAddr")
	ErrRelayVerify             = errors.New("ErrRelayVerify")
	ErrRelayVerifyAddrNotFound = errors.New("ErrRelayVerifyAddrNotFound")
	ErrRelayWaitBlocksErr      = errors.New("ErrRelayWaitBlocks")
	ErrRelayCoinTxHashUsed     = errors.New("ErrRelayCoinTxHashUsed")
	ErrRelayBtcTxTimeErr       = errors.New("ErrRelayBtcTxTimeErr")
	ErrRelayBtcHeadSequenceErr = errors.New("ErrRelayBtcHeadSequenceErr")
	ErrRelayBtcHeadHashErr     = errors.New("ErrRelayBtcHeadHashErr")
	ErrRelayBtcHeadBitsErr     = errors.New("ErrRelayBtcHeadBitsErr")
	ErrRelayBtcHeadNewBitsErr  = errors.New("ErrRelayBtcHeadNewBitsErr")
)
