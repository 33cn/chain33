package types

import "errors"

var (
	ErrRetrieveRepeatAddress   = errors.New("ErrRetrieveRepeatAddress")
	ErrRetrieveDefaultAddress  = errors.New("ErrRetrieveDefaultAddress")
	ErrRetrievePeriodLimit     = errors.New("ErrRetrievePeriodLimit")
	ErrRetrieveAmountLimit     = errors.New("ErrRetrieveAmountLimit")
	ErrRetrieveTimeweightLimit = errors.New("ErrRetrieveTimeweightLimit")
	ErrRetrievePrepareAddress  = errors.New("ErrRetrievePrepareAddress")
	ErrRetrievePerformAddress  = errors.New("ErrRetrievePerformAddress")
	ErrRetrieveCancelAddress   = errors.New("ErrRetrieveCancelAddress")
	ErrRetrieveStatus          = errors.New("ErrRetrieveStatus")
	ErrRetrieveRelateLimit     = errors.New("ErrRetrieveRelateLimit")
	ErrRetrieveRelation        = errors.New("ErrRetrieveRelation")
	ErrRetrieveNoBalance       = errors.New("ErrRetrieveNoBalance")
)
