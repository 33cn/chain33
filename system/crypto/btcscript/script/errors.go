package script

import "errors"

var (

	// ErrInvalidMultiSigRequiredNum error required multi sig pub key num
	ErrInvalidMultiSigRequiredNum = errors.New("ErrInvalidMultiSigRequiredNum")
	// ErrInvalidBtcPubKey invalid bitcoin pubkey
	ErrInvalidBtcPubKey = errors.New("ErrInvalidBtcPubKey")
	// ErrBuildBtcScript build btc script error
	ErrBuildBtcScript = errors.New("ErrBuildBtcScript")
	// ErrNewBtcAddress new btc address pub key error
	ErrNewBtcAddress = errors.New("ErrNewBtcAddress")
	// ErrGetBtcTxInSig get btc tx input signature error
	ErrGetBtcTxInSig = errors.New("ErrGetBtcTxInSig")
	// ErrNewBtcScriptSig new btc script sig error
	ErrNewBtcScriptSig = errors.New("ErrNewBtcScriptSig")

	// ErrBtcKeyNotExist btc key not exist when sign
	ErrBtcKeyNotExist = errors.New("ErrBtcKeyNotExist")
	// ErrBtcScriptNotExist btc script not exist when sign
	ErrBtcScriptNotExist = errors.New("ErrBtcScriptNotExist")
)
