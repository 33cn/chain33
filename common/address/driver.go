// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address

import (
	"errors"
	"sync"
)

var (

	//ErrUnknownAddressDriver 未注册驱动
	ErrUnknownAddressDriver = errors.New("ErrUnknownAddressDriver")
	//ErrAddressDriverNotEnable 驱动未启用
	ErrAddressDriverNotEnable = errors.New("ErrAddressDriverNotEnable")
)

var (
	drivers     = make(map[string]*BaseDriver)
	driverMutex sync.Mutex
)

// Driver address driver
type Driver interface {

	// Format format to address
	Format(pubKey []byte) string
	// Validate address validation
	Validate(addr string) error
}

type BaseDriver struct {
	name         string
	driver       Driver
	enableHeight int64
}

// Register 注册地址驱动
// enableHeight, 设置默认启用高度, 负数表示不启用
func Register(name string, driver Driver, enableHeight int64) {

	driverMutex.Lock()
	defer driverMutex.Unlock()
	_, ok := drivers[name]
	if ok {
		panic("Register duplicate Address driver, name=" + name)
	}
	base := &BaseDriver{
		driver:       driver,
		name:         name,
		enableHeight: enableHeight,
	}
	drivers[name] = base
}

// Load 根据名称加载插件, 根据区块高度判定是否已启动
// 不关心启用状态, blockHeight传-1
func Load(name string, blockHeight int64) (Driver, error) {

	base, ok := drivers[name]
	if !ok {
		return nil, ErrUnknownAddressDriver
	}

	if blockHeight >= 0 && base.enableHeight > blockHeight {
		return nil, ErrAddressDriverNotEnable
	}

	return base.driver, nil
}

// LoadByAddr 根据地址加载驱动
func LoadByAddr(addr string, blockHeight int64) (Driver, error) {

	name := GetDriverName(addr)
	return Load(name, blockHeight)
}

// GetDriverName 基于地址格式, 判定驱动名称
// 每个地址驱动生成的地址格式特征不同, 新增驱动需要调整这个接口
func GetDriverName(addr string) string {

	// 这里遍历所有驱动, 返回第一个合法的, 驱动间如果有相似格式,会导致潜在问题
	for name, base := range drivers {
		if base.driver.Validate(addr) != nil {
			return name
		}
	}
	return ""
}
