// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package types

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(typesCode{})
}

type typesCode struct {
	base.DappCodeFile
}

func (c typesCode) GetDirName() string {

	return "types"
}

func (c typesCode) GetFiles() map[string]string {

	return map[string]string{
		typesName: typesContent,
	}
}

func (c typesCode) GetReplaceTags() []string {

	return []string{types.TagExecName, types.TagClassName,
		types.TagActionIDText, types.TagTyLogActionType,
		types.TagLogMapText, types.TagTypeMapText}
}

var (
	typesName    = "${EXECNAME}.go"
	typesContent = `package types

import (
"encoding/json"

"github.com/33cn/chain33/types"
)

/* 
 * 交易相关类型定义
 * 交易action通常有对应的log结构，用于交易回执日志记录
 * 每一种action和log需要用id数值和name名称加以区分
*/


// action类型id值
${ACTIONIDTEXT}

// log类型id值
${TYLOGACTIONTYPE}

var (
    //${CLASSNAME}X 执行器名称定义
	${CLASSNAME}X = "${EXECNAME}"
	//定义action的name和id
	actionMap = ${TYPEMAPTEXT}
	//定义log的id和具体log类型及名称，填入具体自定义log类型
	logMap = ${LOGMAPTEXT}
)

func init() {
    types.AllowUserExec = append(types.AllowUserExec, []byte(${CLASSNAME}X))
    types.RegistorExecutor(${CLASSNAME}X, newType())
}

type ${EXECNAME}Type struct {
    types.ExecTypeBase
}

func newType() *${EXECNAME}Type {
    c := &${EXECNAME}Type{}
    c.SetChild(c)
    return c
}

// GetPayload 获取合约action结构
func (t *${EXECNAME}Type) GetPayload() types.Message {
    return &${CLASSNAME}Action{}
}

// GeTypeMap 获取合约action的id和name信息
func (t *${EXECNAME}Type) GetTypeMap() map[string]int32 {
    return actionMap
}

// GetLogMap 获取合约log相关信息
func (t *${EXECNAME}Type) GetLogMap() map[int64]*types.LogInfo {
    return logMap
}

// CreateTx 重载基类接口，实现本合约交易创建，供框架调用
func (t *${EXECNAME}Type) CreateTx(action string, message json.RawMessage) (*types.Transaction, error) {
	var tx *types.Transaction
	// pseudo code
	//if action == someAction
		//return new tx
	return tx, types.ErrNotSupport
}

`
)
