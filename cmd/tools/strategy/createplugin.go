//Copyright Fuzamei Corp. 2018 All Rights Reserved.
//Use of this source code is governed by a BSD-style
//license that can be found in the LICENSE file.
package strategy

import "fmt"

// createPluginStrategy 根據模板配置文件，創建一個完整的項目工程
type createPluginStrategy struct {
	strategyBasic
}

func (this *createPluginStrategy) Run() error {
	fmt.Println("Begin run chain33 create plugin project mode.")
	defer fmt.Println("Run chain33 create plugin project mode finish.")
	return this.rumImpl()
}

func (this *createPluginStrategy) rumImpl() error {
	return nil
}
