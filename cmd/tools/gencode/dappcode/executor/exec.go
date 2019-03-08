// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


package executor

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)


func init(){
	base.RegisterCodeFile(execCode{})
	base.RegisterCodeFile(execLocalCode{})
	base.RegisterCodeFile(execDelLocalCode{})
}

type execCode struct {
	executorCodeFile
}


func (execCode)GetFiles() map[string]string {

	return map[string]string{
		execName: executorContent,
	}
}

func (execCode)GetReplaceTags() []string {

	return []string{types.TagExecName, types.TagExecFileContent}
}


type execLocalCode struct {
	executorCodeFile
}

func (execLocalCode)GetFiles() map[string]string {

	return map[string]string{
		execLocalName:execLocalContent,
	}
}

func (execLocalCode)GetReplaceTags() []string {

	return []string{types.TagExecName, types.TagExecLocalFileContent}
}


type execDelLocalCode struct {
	executorCodeFile
}

func (execDelLocalCode)GetFiles() map[string]string {

	return map[string]string{
		execDelName:execDelContent,
	}
}

func (execDelLocalCode)GetReplaceTags() []string {

	return []string{types.TagExecName, types.TagExecDelLocalFileContent}
}





var (

	execName = "exec.go"
	execContent =
`package executor

import (
	ptypes "github.com/33cn/plugin/plugin/dapp/${EXECNAME}/types"
	"github.com/33cn/chain33/types"
)

${EXECFILECONTENT}`

	execLocalName = "exec_local.go"
	execLocalContent =
`package executor

import (
	ptypes "github.com/33cn/plugin/plugin/dapp/${EXECNAME}/types"
	"github.com/33cn/chain33/types"
)

${EXECLOCALFILECONTENT}`


	execDelName = "exec_del_local.go"
	execDelContent =
`package executor

import (
	ptypes "github.com/33cn/plugin/plugin/dapp/${EXECNAME}/types"
	"github.com/33cn/chain33/types"
)

${EXECDELLOCALFILECONTENT}`


)