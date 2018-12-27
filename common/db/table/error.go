// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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
	ErrNilValue               = errors.New("ErrNilValue")
)
