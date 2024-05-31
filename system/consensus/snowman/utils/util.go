package utils

import "errors"

const (
	UnknowMsgID = iota
)

var (
	ErrBlockNotReady   = errors.New("ErrBlockNotReady")
	ErrValidatorSample = errors.New("ErrValidatorSample")
)
