// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package executor

import (
	"fmt"
)

// ConfigPrefix 配置前缀key
var configPrefix = "mavl-config-"

// ConfigKey 原来实现有bug， 但生成的key在状态树里， 不可修改
// mavl-config–{key}  key 前面两个-
func configKey(key string) []byte {
	return []byte(fmt.Sprintf("%s-%s", configPrefix, key))
}

// ManagePrefix 超级管理员账户配置前缀key
var managePrefix = "mavl-manage"

//ManageKey 超级管理员账户key
func manageKey(key string) []byte {
	return []byte(fmt.Sprintf("%s-%s", managePrefix, key))
}

func managerIdKey(id string) []byte {
	return []byte(fmt.Sprintf("%s-%s", managePrefix+"-id", id))
}
