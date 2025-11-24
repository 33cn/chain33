// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package crypto 加解密、签名接口定义
package crypto

import (
	"errors"
	"fmt"
	"sync"
)

var (
	//ErrNotSupportAggr 不支持聚合签名
	ErrNotSupportAggr = errors.New("AggregateCrypto not support")
	//ErrSign 签名错误
	ErrSign = errors.New("error signature")
	//ErrUnknownDriver 未注册加密插件
	ErrUnknownDriver = errors.New("ErrUnknownDriver")
	//ErrDriverNotEnable 加密插件未开启
	ErrDriverNotEnable = errors.New("ErrUnknownDriver")
)

var (
	drivers     = make(map[string]*Driver)
	driversType = make(map[int32]string)
	driverMutex sync.Mutex
)

// Init init crypto
func Init(cfg *Config, subCfg map[string][]byte) {

	if cfg == nil {
		return
	}
	driverMutex.Lock()
	defer driverMutex.Unlock()

	// 未指定时，所有插件是使能的
	if len(cfg.EnableTypes) > 0 {
		//配置中指定了插件类型，先屏蔽所有
		for _, d := range drivers {
			d.enable = false
		}
		// 对配置的插件，默认设置开启高度为0
		for _, name := range cfg.EnableTypes {
			if d, ok := drivers[name]; ok {
				d.enable = true
			}
		}
	}

	// 配置中指定了启用高度，覆盖默认的使能高度设置
	for name, enableHeight := range cfg.EnableHeight {
		// 插件本身在配置中未开启，则enableHeight的配置无效
		if d, ok := drivers[name]; ok && d.enable {
			d.enableHeight = enableHeight
		}
	}

	//插件初始化
	for name, driver := range drivers {
		if driver.initFunc != nil {
			driver.initFunc(subCfg[name])
		}
	}
}

// Register 注册加密算法，支持选项，设置typeID相关参数
func Register(name string, crypto Crypto, options ...RegOption) {
	driverMutex.Lock()
	defer driverMutex.Unlock()

	driver := &Driver{name: name, crypto: crypto, enable: true}

	for _, option := range options {
		if err := option(driver); err != nil {
			panic(err)
		}
	}

	// 未指定情况，系统生成对应的typeID
	if driver.typeID <= 0 {
		driver.typeID = GenDriverTypeID(name)
	}

	d, ok := drivers[name]
	// cgo和go版本同时存在时，允许重复注册
	if ok {
		if driver.isCGO == d.isCGO {
			panic("crypto: Register duplicate driver name, name=" + name)
		}
		// 检测cgo和go版本typeID值是否一致
		if driver.typeID != d.typeID {
			panic(fmt.Sprintf("crypto: Register differt type id in cgo version,"+
				" typeID=[%d, %d]", d.typeID, driver.typeID))
		}
		// 替换go版本，使用性能更好的cgo版本
		if driver.isCGO {
			drivers[name] = driver
		}
		return
	}

	if _, ok := driversType[driver.typeID]; ok {
		// 有重复直接显示报错, 这里有可能是系统自动生成重复TypeID导致的，不能隐式解决重复
		// 因为不同插件组合方式，冲突的情况也不同，需要及时报错并采用手动指定方式解决
		panic(fmt.Sprintf("crypto: Register duplicate driver typeID=%d, "+
			"use WithRegOptionTypeID for manual setting", driver.typeID))
	}
	drivers[name] = driver
	driversType[driver.typeID] = name
}

// GetName 获取name
func GetName(ty int) string {

	name, ok := driversType[int32(ty)]
	if ok {
		return name
	}
	return "unknown"
}

// GetType 获取type
func GetType(name string) int {
	if driver, ok := drivers[name]; ok {
		return int(driver.typeID)
	}
	return 0
}

// Load 加载加密插件, 内部会检测在指定区块高度是否使能
//
// 不考虑使能情况, 只做插件加载, blockHeight传负值, 如-1
func Load(name string, blockHeight int64, opts ...LoadOption) (Crypto, error) {
	if len(opts) > 0 {
		return load(name, append(opts, WithLoadOptionEnableCheck(blockHeight))...)
	}
	return load(name, WithLoadOptionEnableCheck(blockHeight))
}

// load crypto with defined options
func load(name string, opts ...LoadOption) (Crypto, error) {
	c, ok := drivers[name]
	if !ok {
		return nil, ErrUnknownDriver
	}

	for _, opt := range opts {
		if err := opt(c); err != nil {
			return nil, err
		}
	}

	return c.crypto, nil
}

// GetCryptoList 获取加密插件列表，名称和对应的类型值
func GetCryptoList() ([]string, []int32) {

	names := make([]string, 0, len(driversType))
	typeIDs := make([]int32, 0, len(driversType))

	for ty, name := range driversType {
		names = append(names, name)
		typeIDs = append(typeIDs, ty)
	}
	return names, typeIDs
}
