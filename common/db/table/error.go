package table

import "errors"

//table 中的错误处理
var (
	ErrEmptyPrimaryKey        = errors.New("ErrEmptyPrimaryKey")
	ErrPrimaryKey             = errors.New("ErrPrimaryKey")
	ErrIndexKey               = errors.New("ErrIndexKey")
	ErrTooManyIndex           = errors.New("ErrTooManyIndex")
	ErrTablePrefixOrTableName = errors.New("ErrTablePrefixOrTableName")
	ErrDupPrimaryKey          = errors.New("ErrDupPrimaryKey")
)
