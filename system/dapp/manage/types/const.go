// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

// ManageActionModifyConfig manager action
const (
	ManageActionModifyConfig = iota
	ManageActionApplyConfig
	ManageActionApproveConfig
)

// TyLogModifyConfig log
const (
	TyLogModifyConfig  = 410
	TyLogApplyConfig   = 411
	TyLogApproveConfig = 412
)

// ConfigItemArrayConfig config Item
const (
	ConfigItemArrayConfig = iota
)

// ManageConfigStatus config status
const (
	ManageConfigStatusNone     = 0
	ManageConfigStatusApply    = 1
	ManageConfigStatusApproved = 2
)

// OpAdd config op
const (
	OpAdd    = "add"
	OpDelete = "delete"
)
