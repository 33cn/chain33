package types

import "errors"

var (
	ErrTokenNameLen         = errors.New("ErrTokenNameLength")
	ErrTokenSymbolLen       = errors.New("ErrTokenSymbolLength")
	ErrTokenTotalOverflow   = errors.New("ErrTokenTotalOverflow")
	ErrTokenSymbolUpper     = errors.New("ErrTokenSymbolUpper")
	ErrTokenIntroLen        = errors.New("ErrTokenIntroductionLen")
	ErrTokenExist           = errors.New("ErrTokenSymbolExistAlready")
	ErrTokenNotPrecreated   = errors.New("ErrTokenNotPrecreated")
	ErrTokenCreatedApprover = errors.New("ErrTokenCreatedApprover")
	ErrTokenRevoker         = errors.New("ErrTokenRevoker")
	ErrTokenCanotRevoked    = errors.New("ErrTokenCanotRevokedWithWrongStatus")
	ErrTokenOwner           = errors.New("ErrTokenSymbolOwnerNotMatch")
	ErrTokenHavePrecreated  = errors.New("ErrOwnerHaveTokenPrecreateYet")
	ErrTokenBlacklist       = errors.New("ErrTokenBlacklist")
	ErrTokenNotExist        = errors.New("ErrTokenSymbolNotExist")
)
