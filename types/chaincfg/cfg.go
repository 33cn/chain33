// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package chaincfg 实现chain33的基础配置相关功能
package chaincfg

var configMap = make(map[string]string)

// Register 注册配置
func Register(name, cfg string) {
	if _, ok := configMap[name]; ok {
		panic("chain default config name " + name + " is exist")
	}
	configMap[name] = cfg
}

// Load 加载指定配置项
func Load(name string) string {
	return configMap[name]
}

// LoadAll 加载所有配置项
func LoadAll() map[string]string {
	return configMap
}
