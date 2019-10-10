// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)

func init() {
	base.RegisterCodeFile(execCode{})
	base.RegisterCodeFile(execLocalCode{})
	base.RegisterCodeFile(execDelLocalCode{})
}

type execCode struct {
	executorCodeFile
}

func (execCode) GetFiles() map[string]string {

	return map[string]string{
		execName: execContent,
	}
}

func (execCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagClassName, types.TagExecFileContent, types.TagExecObject}
}

type execLocalCode struct {
	executorCodeFile
}

func (execLocalCode) GetFiles() map[string]string {

	return map[string]string{
		execLocalName: execLocalContent,
	}
}

func (execLocalCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagExecLocalFileContent, types.TagExecObject}
}

type execDelLocalCode struct {
	executorCodeFile
}

func (execDelLocalCode) GetFiles() map[string]string {

	return map[string]string{
		execDelName: execDelContent,
	}
}

func (execDelLocalCode) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagExecDelLocalFileContent, types.TagExecObject}
}

var (
	execName    = "exec.go"
	execContent = `package executor

import (
	${EXECNAME}types "${IMPORTPATH}/${EXECNAME}/types"
	"github.com/33cn/chain33/types"
)

/*
 * 实现交易的链上执行接口
 * 关键数据上链（statedb）并生成交易回执（log）
*/

${EXECFILECONTENT}`

	execLocalName    = "exec_local.go"
	execLocalContent = `package executor

import (
	${EXECNAME}types "${IMPORTPATH}/${EXECNAME}/types"
	"github.com/33cn/chain33/types"
)

/*
 * 实现交易相关数据本地执行，数据不上链
 * 非关键数据，本地存储(localDB), 用于辅助查询，效率高
*/

${EXECLOCALFILECONTENT}

//设置自动回滚
func (${EXEC_OBJECT} *${EXECNAME}) addAutoRollBack(tx *types.Transaction, kv []*types.KeyValue) *types.LocalDBSet {

	dbSet := &types.LocalDBSet{}
	dbSet.KV = ${EXEC_OBJECT}.AddRollbackKV(tx, tx.Execer, kv)
	return dbSet
}
`

	execDelName    = "exec_del_local.go"
	execDelContent = `package executor

import (
	"github.com/33cn/chain33/types"
)

/* 
 * 实现区块回退时本地执行的数据清除
*/

// ExecDelLocal 回退自动删除，重写基类
func (${EXEC_OBJECT} *${EXECNAME}) ExecDelLocal(tx *types.Transaction, receipt *types.ReceiptData, index int) (*types.LocalDBSet, error) {
	kvs, err := ${EXEC_OBJECT}.DelRollbackKV(tx, tx.Execer)
	if err != nil {
		return nil, err
	}
	dbSet := &types.LocalDBSet{}
	dbSet.KV = append(dbSet.KV, kvs...)
	return dbSet, nil
}
`
)
