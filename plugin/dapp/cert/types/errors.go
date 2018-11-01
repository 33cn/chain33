package types

import "errors"

var (
	ErrValidateCertFailed  = errors.New("ErrValidateCertFailed")
	ErrGetHistoryCertData  = errors.New("ErrGetHistoryCertData")
	ErrUnknowAuthSignType  = errors.New("ErrUnknowAuthSignType")
	ErrInitializeAuthority = errors.New("ErrInitializeAuthority")
)
