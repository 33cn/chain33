package types

import "errors"

var (
	ErrNoPrivilege              = errors.New("ErrNoPrivilege")
	ErrLotteryStatus            = errors.New("ErrLotteryStatus")
	ErrLotteryDrawActionInvalid = errors.New("ErrLotteryDrawActionInvalid")
	ErrLotteryFundNotEnough     = errors.New("ErrLotteryFundNotEnough")
	ErrLotteryCreatorBuy        = errors.New("ErrLotteryCreatorBuy")
	ErrLotteryBuyAmount         = errors.New("ErrLotteryBuyAmount")
	ErrLotteryRepeatHash        = errors.New("ErrLotteryRepeatHash")
	ErrLotteryPurBlockLimit     = errors.New("ErrLotteryPurBlockLimit")
	ErrLotteryDrawBlockLimit    = errors.New("ErrLotteryDrawBlockLimit")
	ErrLotteryBuyNumber         = errors.New("ErrLotteryBuyNumber")
	ErrLotteryShowRepeated      = errors.New("ErrLotteryShowRepeated")
	ErrLotteryShowError         = errors.New("ErrLotteryShowError")
	ErrLotteryErrLuckyNum       = errors.New("ErrLotteryErrLuckyNum")
	ErrLotteryErrCloser         = errors.New("ErrLotteryErrCloser")
	ErrLotteryErrUnableClose    = errors.New("ErrLotteryErrUnableClose")
	ErrNodeNotExist             = errors.New("ErrNodeNotExist")
	ErrEmptyMinerTx             = errors.New("ErrEmptyMinerTx")
)
