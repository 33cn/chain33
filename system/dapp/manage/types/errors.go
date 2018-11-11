// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "errors"

var (
	ErrNoPrivilege    = errors.New("ErrNoPrivilege")
	ErrBadConfigKey   = errors.New("ErrBadConfigKey")
	ErrBadConfigOp    = errors.New("ErrBadConfigOp")
	ErrBadConfigValue = errors.New("ErrBadConfigValue")
)
