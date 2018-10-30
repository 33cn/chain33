package types

import "errors"

var (
	ErrNoPrivilege    = errors.New("ErrNoPrivilege")
	ErrBadConfigKey   = errors.New("ErrBadConfigKey")
	ErrBadConfigOp    = errors.New("ErrBadConfigOp")
	ErrBadConfigValue = errors.New("ErrBadConfigValue")
)
