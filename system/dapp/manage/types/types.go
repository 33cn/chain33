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
		"Modify":  ManageActionModifyConfig,
		"Apply":   ManageActionApplyConfig,
		"Approve": ManageActionApproveConfig,
	}
	logmap = map[int64]*types.LogInfo{
		// 这里reflect.TypeOf类型必须是proto.Message类型，且是交易的回持结构
		TyLogModifyConfig:  {Ty: reflect.TypeOf(types.ReceiptConfig{}), Name: "LogModifyConfig"},
		TyLogApplyConfig:   {Ty: reflect.TypeOf(ReceiptApplyConfig{}), Name: "LogApplyConfig"},
		TyLogApproveConfig: {Ty: reflect.TypeOf(ReceiptApproveConfig{}), Name: "LogApproveConfig"},
	}
)

const (
	//ForkManageExec manage key
	ForkManageExec = "ForkManageExec"
	//ForkManageAutonomyEnable enable approve from autonomy
	ForkManageAutonomyEnable = "ForkManageAutonomyEnable"
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(ManageX))
	types.RegFork(ManageX, InitFork)
	types.RegExec(ManageX, InitExecutor)
}

// InitFork init
func InitFork(cfg *types.Chain33Config) {
	cfg.RegisterDappFork(ManageX, "Enable", 0)
	cfg.RegisterDappFork(ManageX, ForkManageExec, 0)
	//支持autonomy委员会审批
	cfg.RegisterDappFork(ManageX, ForkManageAutonomyEnable, 0)
}

// InitExecutor init Executor
func InitExecutor(cfg *types.Chain33Config) {
	types.RegistorExecutor(ManageX, NewType(cfg))
}

// ManageType defines managetype
type ManageType struct {
	types.ExecTypeBase
}

// NewType new a managetype object
func NewType(cfg *types.Chain33Config) *ManageType {
	c := &ManageType{}
	c.SetChild(c)
	c.SetConfig(cfg)
	return c
}

// GetPayload return manageaction
func (m *ManageType) GetPayload() types.Message {
	return &ManageAction{}
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
