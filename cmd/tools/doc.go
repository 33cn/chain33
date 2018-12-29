// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
Package main chain33开发者工具，主要提供以下功能：
1. 通过chain33.cpm.toml配置，指定需要下载的包，从远程下载到本地 import
2. 通过本地创建各种执行器工程，相关命令为 simple, advance
3. 扫描本地插件信息，更新引用关系
4. 通过本地创建完整的插件项目,可以选择simple模式和advance模式.

目录介绍
	1. config目录为tools工具使用的配置目录
	2. config/chain33.cpm.toml 是tools工具通过go vendor下载三方系统插件的配置
	3. config/exec_header.template 是tools工具创建执行器过程中使用的代码模板
	4. config/types_content.template 是tools工具创建执行器过程中使用的代码模板
	5. config/template 目录是tools工具创建执行器过程中使用的代码模板

库包获取的步骤

简单执行器工程向导

高级执行器工程向导
	文字替换规则：
		${PROJECTNAME}:	设定的项目名称
		${CLASSNAME}:	设定的执行器类名
		${ACTIONNAME}:	执行器内部逻辑使用的
		${EXECNAME}:	执行器的名称
		${TYPENAME}:
	自动创建文件：
		exec.go : 执行器功能中
		exec_local.go:
		exec_del_local.go
	使用步骤：
		1. 按照proto3的语法格式，创建执行器使用的Action结构,参考结构如下
		// actions
		message DemoAction {
			oneof value {
				DemoCreate	create      = 1;
				DemoRun		play        = 2;
				DemoClose	show        = 3;
			}
			int32 ty = 6;
		}
		2. 实现Action中使用的所有类型，例如上面的DemoCreate、DemoRun、DemoClose
		3. 将编辑好的协议文件保存到tools所在目录下的config内
		4. 使用命令行生成
	命令行说明：
		示例：tools advance -n demo
			-a  --action  		执行器中Action使用的类型名，如果不填则为执行器名称+Action，例如DemoAction
			-n  --name    		执行器的项目名和类名，必填参数
			-p  --propfile 		导入执行器类型的proto3协议模板，如果不填默认为config/执行器名称.proto
			-t  --templatepath 	生成执行器项目的模板文件，不填默认为config/template下的所有文件

更新初始化文件:
	扫描指定path目录下所有的插件，根据扫描到的结果重新更新consensus、dapp和、store、mempool的初始化文件 init.go
	使用方式：./tools updateinit -p $(YourPluginPath)
	例子：./tools updateinit -p /GOPATH/src/github.com/33cn/chain33/cmd/tools/plugin

*/
package main
