package types

import "errors"

var (
	ErrInvalidTitle         = errors.New("ErrInvalidTitle")
	ErrTitleNotExist        = errors.New("ErrTitleNotExist")
	ErrNodeNotForTheTitle   = errors.New("ErrNodeNotForTheTitle")
	ErrParaBlockHashNoMatch = errors.New("ErrParaBlockHashNoMatch")
	ErrParaMinerBaseIndex   = errors.New("ErrParaMinerBaseIndex")
	ErrParaMinerTxType      = errors.New("ErrParaMinerTxType")
	ErrParaEmptyMinerTx     = errors.New("ErrParaEmptyMinerTx")
	ErrParaMinerExecErr     = errors.New("ErrParaMinerExecErr")
)
