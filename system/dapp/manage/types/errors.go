// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import "errors"

var (
	// ErrNoPrivilege defines a error string errnoprivilege
	ErrNoPrivilege = errors.New("ErrNoPrivilege")
	// ErrBadConfigKey defines a err string errbadconfigkey
	ErrBadConfigKey = errors.New("ErrBadConfigKey")
	// ErrBadConfigOp defines a err string errbadconfigop
	ErrBadConfigOp = errors.New("ErrBadConfigOp")
	// ErrBadConfigValue defines a err string errbadconfigvalue
	ErrBadConfigValue = errors.New("ErrBadConfigValue")
)
