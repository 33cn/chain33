package table

import "errors"

//table 中的错误处理
var (
	ErrEmptyPrimaryKey = errors.New("ErrEmptyPrimaryKey")
	ErrTooManyIndex    = errors.New("ErrTooManyIndex")
)
