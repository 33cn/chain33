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

// 插件的具体数值类型定义在一个文件中，方便查阅
const (
	TySecp256K1 = 1
	TyEd25519   = 2
	TySm2       = 3
	TyNone      = 10
)

var (
	//ErrNotSupportAggr 不支持聚合签名
	ErrNotSupportAggr = errors.New("AggregateCrypto not support")
	//ErrSign 签名错误
	ErrSign = errors.New("error signature")
)

var (
	drivers       = make(map[string]Crypto)
	driversCGO    = make(map[string]Crypto)
	driversType   = make(map[string]int)
	driverInitFns = make(map[string]DriverInitFn)
	driverMutex   sync.Mutex
)

// Init init crypto
func Init(cfg *Config, subCfg map[string][]byte) {

	//初始化子配置
	for name, fn := range driverInitFns {
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

//RegisterType 注册类型
func RegisterType(name string, ty int) {
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
}

// RegisterDriverInitFn 某些签名插件需要初始化操作，如证书导入，需要将初始化接口进行预先注册
// 配置文件格式[crypto.sub.name]
func RegisterDriverInitFn(name string, fn DriverInitFn) {
	driverMutex.Lock()
	defer driverMutex.Unlock()
	if _, dup := driverInitFns[name]; dup {
		panic("crypto: RegisterDriverInitFn called twice for name: " + name)
	}
	driverInitFns[name] = fn
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
func New(name string) (c Crypto, err error) {

	//优先使用性能更好的cgo版本
	c, ok := driversCGO[name]
	if ok {
		return c, nil
	}
	//不存在cgo, 加载普通版本
	c, ok = drivers[name]
	if !ok {
		err = fmt.Errorf("unknown driver %q", name)
	}
	return c, err
}
