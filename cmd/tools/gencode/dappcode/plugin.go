// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dappcode

import (
	"github.com/33cn/chain33/cmd/tools/gencode/base"
	"github.com/33cn/chain33/cmd/tools/types"
)

func init() {

	base.RegisterCodeFile(pluginCodeFile{})
}

type pluginCodeFile struct {
	base.DappCodeFile
}

func (c pluginCodeFile) GetFiles() map[string]string {

	return map[string]string{
		pluginName: pluginContent,
	}
}

func (c pluginCodeFile) GetFileReplaceTags() []string {

	return []string{types.TagExecName, types.TagImportPath, types.TagClassName}
}

var (
	pluginName    = "plugin.go"
	pluginContent = `
package ${EXECNAME}

import (
	"${IMPORTPATH}/${EXECNAME}/commands"
	ptypes "${IMPORTPATH}/${EXECNAME}/types/${EXECNAME}"
	"${IMPORTPATH}/${EXECNAME}/executor"
	"${IMPORTPATH}/${EXECNAME}/rpc"
	"github.com/33cn/chain33/pluginmgr"
)

/*  
 * 初始化dapp相关的组件
*/

func init() {
	pluginmgr.Register(&pluginmgr.PluginBase{
		Name:     ptypes.${CLASSNAME}X,
		ExecName: executor.GetName(),
		Exec:     executor.Init,
		Cmd:      commands.Cmd,
		RPC:      rpc.Init,
	})
}`
)
