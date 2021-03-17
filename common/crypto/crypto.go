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

// 插件的名称和类型值
const (
	TySecp256K1   = 1
	TyEd25519     = 2
	TySm2         = 3
	TyNone        = 10
	NameSecp256K1 = "secp256k1"
	NameEd25519   = "ed25519"
	NameSm2       = "sm2"
	NameNone      = "none"
)

var (
	//ErrNotSupportAggr 不支持聚合签名
	ErrNotSupportAggr = errors.New("AggregateCrypto not support")
	//ErrSign 签名错误
	ErrSign = errors.New("error signature")
)

var (
	drivers             = make(map[string]Crypto)
	driversCGO          = make(map[string]Crypto)
	driversType         = make(map[string]int)
	driversInitFn       = make(map[string]DriverInitFn)
	driversEnableHeight = make(map[string]int64)
	driverMutex         sync.Mutex
)

// Init init crypto
func Init(cfg *Config, subCfg map[string][]byte) {

	driverMutex.Lock()
	defer driverMutex.Unlock()

	if len(cfg.EnableTypes) > 0 {
		driversEnableHeight = make(map[string]int64, len(cfg.EnableTypes))
		// 对配置的插件，默认设置开启高度为0
		for _, name := range cfg.EnableTypes {
			driversEnableHeight[name] = 0
		}
	}

	// 配置中指定了启用高度，覆盖设置
	for name, enableHeight := range cfg.EnableHeight {
		if _, ok := driversEnableHeight[name]; ok {
			driversEnableHeight[name] = enableHeight
		}
	}

	//初始化子配置 [crypto.sub.name]
	for name, fn := range driversInitFn {
		fn(subCfg[name])
	}
}

//Register 注册加密算法，允许同种加密算法的cgo版本同时注册
func Register(name string, driver Crypto, isCGO bool) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	d := drivers
	if isCGO {
		d = driversCGO
	}
	if _, dup := d[name]; dup {
		panic("crypto: Register called twice for driver " + name)
	}
	d[name] = driver
}

//RegisterType 注册类型, 并设置启用高度
func RegisterType(name string, ty int, enableHeight int64) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	for n, t := range driversType {
		//由于可能存在cgo版本，允许name，ty都相等情况，重复注册
		//或者都不相等，添加新的类型
		//不允许只有一个相等的情况，即不允许修改已经存在的ty或者name
		if (n == name && t != ty) || (n != name && t == ty) {
			panic(fmt.Sprintf("crypto: Register Conflict, exist=(%s,%d), register=(%s, %d)", n, t, name, ty))
		}
	}
	driversType[name] = ty
	//设置默认启用高度
	driversEnableHeight[name] = enableHeight
}

// RegisterDriverInitFn 某些签名插件需要初始化操作，如证书导入，需要将初始化接口进行预先注册
// 配置文件格式[crypto.sub.name]
func RegisterDriverInitFn(name string, fn DriverInitFn) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	if _, dup := driversInitFn[name]; dup {
		panic("crypto: RegisterDriverInitFn called twice for name: " + name)
	}
	driversInitFn[name] = fn
}

//GetName 获取name
func GetName(ty int) string {
	for name, t := range driversType {
		if t == ty {
			return name
		}
	}
	return "unknown"
}

//GetType 获取type
func GetType(name string) int {
	if ty, ok := driversType[name]; ok {
		return ty
	}
	return 0
}

//New new
func New(name string) (Crypto, error) {

	//优先使用性能更好的cgo版本
	c, ok := driversCGO[name]
	if ok {
		return c, nil
	}
	//不存在cgo, 加载普通版本
	c, ok = drivers[name]
	if !ok {
		return nil, fmt.Errorf("unknown driver %q", name)
	}
	return c, nil
}

// IsEnable 根据高度判定是否开启
func IsEnable(name string, height int64) bool {

	enableHeight, ok := driversEnableHeight[name]
	return ok && enableHeight >= 0 && enableHeight <= height
}
