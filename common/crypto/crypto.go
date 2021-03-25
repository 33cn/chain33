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

	// 未指定时，所有插件采用代码中的开启配置
	if len(cfg.EnableTypes) > 0 {
		//配置中指定了插件类型，先屏蔽所有
		for _, d := range drivers {
			d.enableHeight = -1
		}
		// 对配置的插件，默认设置开启高度为0
		for _, name := range cfg.EnableTypes {
			if d, ok := drivers[name]; ok {
				d.enableHeight = 0
			}
		}
	}

	// 配置中指定了启用高度，覆盖设置
	for name, enableHeight := range cfg.EnableHeight {
		// enableHeight如果为-1，表示插件本身在配置中未开启，则enableHeight的配置无效
		if d, ok := drivers[name]; ok && d.enableHeight >= 0 {
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

//Register 注册加密算法，支持选项，设置typeID相关参数
func Register(name string, crypto Crypto, options ...Option) {
	driverMutex.Lock()
	defer driverMutex.Unlock()

	driver := &Driver{crypto: crypto}

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
			"use WithOptionTypeID for manual setting", driver.typeID))
	}
	drivers[name] = driver
	driversType[driver.typeID] = name
}

//GetName 获取name
func GetName(ty int) string {

	name, ok := driversType[int32(ty)]
	if ok {
		return name
	}
	return "unknown"
}

//GetType 获取type
func GetType(name string) int {
	if driver, ok := drivers[name]; ok {
		return int(driver.typeID)
	}
	return 0
}

//New new
func New(name string) (Crypto, error) {

	c, ok := drivers[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver %q", name)
	}
	return c.crypto, nil
}

// IsEnable 根据高度判定是否开启
func IsEnable(name string, height int64) bool {

	d, ok := drivers[name]
	return ok && d.enableHeight >= 0 && d.enableHeight <= height
}
