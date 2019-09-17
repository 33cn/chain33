// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types 管理插件相关的定义
package types

import (
	"reflect"

	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/types"
)

var (
	// ManageX defines a global string
	ManageX    = "manage"
	actionName = map[string]int32{
		"Modify": ManageActionModifyConfig,
	}
	logmap = map[int64]*types.LogInfo{
		// 这里reflect.TypeOf类型必须是proto.Message类型，且是交易的回持结构
		TyLogModifyConfig: {Ty: reflect.TypeOf(types.ReceiptConfig{}), Name: "LogModifyConfig"},
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(ManageX))
	types.RegistorExecutor(ManageX, NewType())

	types.RegisterDappFork(ManageX, "Enable", 120000)
	types.RegisterDappFork(ManageX, "ForkManageExec", 400000)
}

// ManageType defines managetype
type ManageType struct {
	types.ExecTypeBase
}

// NewType new a managetype object
func NewType() *ManageType {
	c := &ManageType{}
	c.SetChild(c)
	return c
}

// GetPayload return manageaction
func (m *ManageType) GetPayload() types.Message {
	return &ManageAction{}
}

// ActionName return action a string name
func (m ManageType) ActionName(tx *types.Transaction) string {
	return "config"
}

// Amount amount
func (m ManageType) Amount(tx *types.Transaction) (int64, error) {
	return 0, nil
}

// GetLogMap  get log for map
func (m *ManageType) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

// GetRealToAddr main reason for overloading this function is because of the manage protocol
// during implementation, to address specification varies fron height to height
func (m ManageType) GetRealToAddr(tx *types.Transaction) string {
	if len(tx.To) == 0 {
		// 如果To地址为空，则认为是早期低于types.ForkV11ManageExec高度的交易，直接返回合约地址
		return address.ExecAddress(string(tx.Execer))
	}
	return tx.To
}

// GetTypeMap return typename of actionname
func (m ManageType) GetTypeMap() map[string]int32 {
	return actionName
}

// GetName reset name
func (m *ManageType) GetName() string {
	return ManageX
}
