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


func (c typesCode)GetDirName() string {

	return "types"
}


func (c typesCode)GetFiles() map[string]string {

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

	typesName = "${EXECNAME}.go"
	typesContent =
`package types

import (
"github.com/33cn/chain33/types"
)

// action for executor
${ACTIONIDTEXT}

${TYLOGACTIONTYPE}

var (
    
	${CLASSNAME}X = "${EXECNAME}"
	logMap = ${LOGMAPTEXT}
    typeMap = ${TYPEMAPTEXT}
)

func init() {
    types.AllowUserExec = append(types.AllowUserExec, []byte(${EXECNAME}))
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

func (t *${EXECNAME}Type) GetPayload() types.Message {
    return &${EXECNAME}Action{}
}

func (t *${EXECNAME}Type) GetTypeMap() map[string]int32 {
    return typeMap
}

func (t *${EXECNAME}Type) GetLogMap() map[int64]*types.LogInfo {
    return logMap
}

`
)