// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(executorCodeFile{})
}

type executorCodeFile struct {
	base.DappCodeFile
}

func (c executorCodeFile) GetDirName() string {

	return "executor"
}

func (c executorCodeFile) GetFiles() map[string]string {

	return map[string]string{
		executorName: executorContent,
		kvName:       kvContent,
	}
}

func (c executorCodeFile) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagClassName}
}

var (
	executorName    = "${EXECNAME}.go"
	executorContent = `package executor

import (
	log "github.com/33cn/chain33/common/log/log15"
	ptypes "${IMPORTPATH}/${EXECNAME}/types/${EXECNAME}"
	drivers "github.com/33cn/chain33/system/dapp"
	"github.com/33cn/chain33/types"
)

/* 
 * 执行器相关定义
 * 重载基类相关接口
*/ 


var (
	//日志
	elog = log.New("module", "${EXECNAME}.executor")
)

var driverName = ptypes.${CLASSNAME}X

func init() {
	ety := types.LoadExecutorType(driverName)
	ety.InitFuncList(types.ListMethod(&${EXECNAME}{}))
}

// Init register dapp
func Init(name string, sub []byte) {
	drivers.Register(GetName(), new${CLASSNAME}, types.GetDappFork(driverName, "Enable"))
}

type ${EXECNAME} struct {
	drivers.DriverBase
}

func new${CLASSNAME}() drivers.Driver {
	t := &${EXECNAME}{}
	t.SetChild(t)
	t.SetExecutorType(types.LoadExecutorType(driverName))
	return t
}

// GetName get driver name
func GetName() string {
	return new${CLASSNAME}().GetName()
}

func (*${EXECNAME}) GetDriverName() string {
	return driverName
}

// CheckTx 实现自定义检验交易接口，供框架调用
func (*${EXECNAME}) CheckTx(tx *types.Transaction, index int) error {
	// implement code
	return nil
}

`

	kvName    = "kv.go"
	kvContent = `package executor

/*
 * 用户合约存取kv数据时，key值前缀需要满足一定规范
 * 即key = keyPrefix + userKey
 * 需要字段前缀查询时，使用’-‘作为分割符号
*/

var (
	//KeyPrefixStateDB state db key必须前缀
	KeyPrefixStateDB = "mavl-${EXECNAME}-"
	//KeyPrefixLocalDB local db的key必须前缀
	KeyPrefixLocalDB = "LODB-${EXECNAME}-"
)
`
)
