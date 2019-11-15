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

func (c typesCode) GetDirReplaceTags() []string {
	return []string{types.TagExecName}
}

func (c typesCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagExecObject, types.TagClassName,
		types.TagActionIDText, types.TagTyLogActionType,
		types.TagLogMapText, types.TagTypeMapText}
}

var (
	typesName    = "${EXECNAME}.go"
	typesContent = `package types

import (
"encoding/json"
log "github.com/33cn/chain33/common/log/log15"
"github.com/33cn/chain33/types"
)

/* 
 * 交易相关类型定义
 * 交易action通常有对应的log结构，用于交易回执日志记录
 * 每一种action和log需要用id数值和name名称加以区分
*/


// action类型id和name，这些常量可以自定义修改
${ACTIONIDTEXT}

// log类型id值
${TYLOGACTIONTYPE}

var (
    //${CLASSNAME}X 执行器名称定义
	${CLASSNAME}X = "${EXECNAME}"
	//定义actionMap
	actionMap = ${TYPEMAPTEXT}
	//定义log的id和具体log类型及名称，填入具体自定义log类型
	logMap = ${LOGMAPTEXT}
	tlog = log.New("module", "${EXECNAME}.types")
)

// init defines a register function
func init() {
    types.AllowUserExec = append(types.AllowUserExec, []byte(${CLASSNAME}X))
	//注册合约启用高度
	types.RegFork(${CLASSNAME}X, InitFork)
	types.RegExec(${CLASSNAME}X, InitExecutor)
}

// InitFork defines register fork
func InitFork(cfg *types.Chain33Config) {
	cfg.RegisterDappFork(${CLASSNAME}X, "Enable", 0)
}

// InitExecutor defines register executor
func InitExecutor(cfg *types.Chain33Config) {
	types.RegistorExecutor(${CLASSNAME}X, NewType(cfg))
}

type ${EXECNAME}Type struct {
    types.ExecTypeBase
}

func NewType(cfg *types.Chain33Config) *${EXECNAME}Type {
    c := &${EXECNAME}Type{}
    c.SetChild(c)
    c.SetConfig(cfg)
    return c
}

// GetPayload 获取合约action结构
func (${EXEC_OBJECT} *${EXECNAME}Type) GetPayload() types.Message {
    return &${CLASSNAME}Action{}
}

// GeTypeMap 获取合约action的id和name信息
func (${EXEC_OBJECT} *${EXECNAME}Type) GetTypeMap() map[string]int32 {
    return actionMap
}

// GetLogMap 获取合约log相关信息
func (${EXEC_OBJECT} *${EXECNAME}Type) GetLogMap() map[int64]*types.LogInfo {
    return logMap
}

`
)
