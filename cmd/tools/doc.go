// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*

更新初始化文件:
	扫描指定path目录下所有的插件，根据扫描到的结果重新更新consensus、dapp和、store、mempool的初始化文件 init.go
	使用方式：./tools updateinit -p $(YourPluginPath)
	例子：./tools updateinit -p /GOPATH/src/github.com/33cn/chain33/cmd/tools/plugin

*/
package main
