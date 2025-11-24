// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package types 管理插件相关的定义
package types

import (
	"reflect"

	"github.com/33cn/chain33/types"
)

// action类型id和name，这些常量可以自定义修改
const (

	// TyCommitDelayTxAction commit delay transaction action id
	TyCommitDelayTxAction = iota + 101

	// UnknownActionName 存证类型交易
	UnknownActionName = "notary"
	// NameCommitDelayTxAction commit delay transaction action name
	NameCommitDelayTxAction = "CommitDelayTx"
)

// log类型id值
const (
	// TyCommitDelayTxLog commit delay transaction log  id
	TyCommitDelayTxLog = iota + 100

	// NameCommitDelayTxLog commit delay transaction log name
	NameCommitDelayTxLog = "CommitDelayTxLog"
)

// query func name
const (

	// QueryGetDelayTxInfo query func name
	QueryGetDelayTxInfo = "GetDelayTxInfo"
)

// fork
const (
	// ForkUseTimeDelay use block time as delay type, instead of block height
	ForkUseTimeDelay = "ForkUseTimeDelay"
)

var (
	// NoneX driver name
	NoneX      = "none"
	actionName = map[string]int32{
		NameCommitDelayTxAction: TyCommitDelayTxAction,
	}
	logmap = map[int64]*types.LogInfo{

		TyCommitDelayTxLog: {Ty: reflect.TypeOf(CommitDelayTxLog{}), Name: NameCommitDelayTxLog},
	}
)

func init() {
	types.AllowUserExec = append(types.AllowUserExec, []byte(NoneX))
	types.RegFork(NoneX, InitFork)
	types.RegExec(NoneX, InitExecutor)

}

// InitFork init
func InitFork(cfg *types.Chain33Config) {
	cfg.RegisterDappFork(NoneX, ForkUseTimeDelay, 0)
}

// InitExecutor init Executor
func InitExecutor(cfg *types.Chain33Config) {
	types.RegistorExecutor(NoneX, NewType(cfg))
}

// NoneType defines NoneType
type NoneType struct {
	types.ExecTypeBase
}

// NewType new a NoneType object
func NewType(cfg *types.Chain33Config) *NoneType {
	c := &NoneType{}
	c.SetChild(c)
	c.SetConfig(cfg)
	return c
}

// GetPayload return manageaction
func (n *NoneType) GetPayload() types.Message {
	return &NoneAction{}
}

// DecodePayloadValue decode payload value
func (n *NoneType) DecodePayloadValue(tx *types.Transaction) (string, reflect.Value, error) {
	name, value, err := decodeNonePayload(tx)
	return name, value, err
}

// ActionName return action a string name
func (n *NoneType) ActionName(tx *types.Transaction) string {
	return ActionName(tx)
}

// ActionName return action a string name
func ActionName(tx *types.Transaction) string {
	name, _, _ := decodeNonePayload(tx)
	return name
}

// decodeNonePayload decode payload
func decodeNonePayload(tx *types.Transaction) (string, reflect.Value, error) {

	action := &NoneAction{}
	err := types.Decode(tx.Payload, action)
	if err != nil {
		return UnknownActionName, reflect.ValueOf(nil), types.ErrActionNotSupport
	}

	if action.Ty == TyCommitDelayTxAction {
		return NameCommitDelayTxAction, reflect.ValueOf(action.GetCommitDelayTx()), nil
	}

	return UnknownActionName, reflect.ValueOf(nil), types.ErrActionNotSupport
}

// GetLogMap  get log for map
func (n *NoneType) GetLogMap() map[int64]*types.LogInfo {
	return logmap
}

// GetTypeMap return typename of actionname
func (n NoneType) GetTypeMap() map[string]int32 {
	return actionName
}

// GetName reset name
func (n *NoneType) GetName() string {
	return NoneX
}
