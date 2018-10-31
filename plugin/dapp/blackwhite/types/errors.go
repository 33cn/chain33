package types

import "errors"

var (
	ErrIncorrectStatus  = errors.New("ErrIncorrectStatus")
	ErrRepeatPlayerAddr = errors.New("ErrRepeatPlayerAddress")
	ErrNoTimeoutDone    = errors.New("ErrNoTimeoutDone")
	ErrNoExistAddr      = errors.New("ErrNoExistAddress")
	ErrNoLoopSeq        = errors.New("ErrBlackwhiteFinalloopLessThanSeq")
)
