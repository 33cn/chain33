// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package address

import (
	"errors"
	"fmt"
	"sync"
)

var (

	//ErrUnknownAddressDriver 未注册驱动
	ErrUnknownAddressDriver = errors.New("ErrUnknownAddressDriver")
	//ErrUnknownAddressType unknown address type
	ErrUnknownAddressType = errors.New("ErrUnknownAddressType")
	//ErrAddressDriverNotEnable 驱动未启用
	ErrAddressDriverNotEnable = errors.New("ErrAddressDriverNotEnable")
)

var (
	drivers          = make(map[int32]*DriverInfo)
	driverName       = make(map[string]int32)
	defaultAddressID int32 // btc address format as default
	driverMutex      sync.Mutex
)

// Driver address driver
type Driver interface {

	// PubKeyToAddr public key to address
	PubKeyToAddr(pubKey []byte) string
	// ValidateAddr address validation
	ValidateAddr(addr string) error
	// GetName get driver name
	GetName() string
	// FromString decode from string
	FromString(addr string) ([]byte, error)
	// ToString encode to string
	ToString(addr []byte) string
	// FormatAddr to unified format
	FormatAddr(addr string) string
}

// DriverInfo driver info
type DriverInfo struct {
	driver       Driver
	enableHeight int64
}

const (
	// MaxID 最大id值
	MaxID = 7
	// DefaultID default id flag
	DefaultID = -1
)

// Init init with config
func Init(config *Config) {

	if config == nil {
		return
	}
	driverMutex.Lock()
	defer driverMutex.Unlock()

	for name, enableHeight := range config.EnableHeight {

		id, ok := driverName[name]
		if !ok {
			panic(fmt.Sprintf("config address enable height, unknown driver \"%s\"", name))
		}
		drivers[id].enableHeight = enableHeight
	}

	// set default value
	if config.DefaultDriver == "" {
		config.DefaultDriver = drivers[defaultAddressID].driver.GetName()
	}

	defaultID, ok := driverName[config.DefaultDriver]
	if !ok {
		panic(fmt.Sprintf("config default driver, unknown driver \"%s\"", config.DefaultDriver))
	}

	info := drivers[defaultID]
	if info.enableHeight != 0 {
		panic(fmt.Sprintf("default driver \"%s\" enable height should be 0", config.DefaultDriver))
	}
	defaultAddressID = defaultID
}

// RegisterDriver 注册地址驱动
// enableHeight, 设置默认启用高度, 负数表示不启用
func RegisterDriver(id int32, driver Driver, enableHeight int64) {

	driverMutex.Lock()
	defer driverMutex.Unlock()
	if !IsValidAddressID(id) {
		panic(fmt.Sprintf("address id must in range [0, %d]", MaxID))
	}
	_, ok := drivers[id]
	if ok {
		panic(fmt.Sprintf("Register duplicate Address id %d", id))
	}

	_, ok = driverName[driver.GetName()]
	if ok {
		panic(fmt.Sprintf("Register duplicate driver name %s", driver.GetName()))
	}

	info := &DriverInfo{
		driver:       driver,
		enableHeight: enableHeight,
	}
	drivers[id] = info
	driverName[driver.GetName()] = id
}

// MustLoadDriver 根据ID加载插件, 出错panic
func MustLoadDriver(id int32) Driver {

	d, ok := drivers[id]
	if !ok {
		panic(fmt.Sprintf("unknown address driver, id=%d", id))
	}
	return d.driver
}

// LoadDriver 根据ID加载插件, 根据区块高度判定是否已启动
// 不关心启用状态, blockHeight传-1
func LoadDriver(id int32, blockHeight int64) (Driver, error) {

	d, ok := drivers[id]
	if !ok {
		return nil, ErrUnknownAddressDriver
	}

	if !isEnable(blockHeight, d.enableHeight) {
		return nil, ErrAddressDriverNotEnable
	}

	return d.driver, nil
}

// IsValidAddressID is valid
func IsValidAddressID(id int32) bool {
	return id >= 0 && id <= MaxID
}

// GetDefaultAddressID get default id
func GetDefaultAddressID() int32 {
	return defaultAddressID
}

// GetDefaultAddressDriver get default driver
func GetDefaultAddressDriver() Driver {
	d, _ := LoadDriver(defaultAddressID, -1)
	return d
}

// check driver enable height with block height
func isEnable(blockHeight, enableHeight int64) bool {

	if blockHeight < 0 {
		return true
	}
	if enableHeight < 0 || enableHeight > blockHeight {
		return false
	}
	return true
}

// GetDriverList get driver list
func GetDriverList() map[int32]Driver {

	list := make(map[int32]Driver, len(drivers))
	for id, d := range drivers {
		list[id] = d.driver
	}
	return list
}

// GetDriverType get type by name
func GetDriverType(name string) (int32, error) {
	ty, ok := driverName[name]
	if !ok {
		return -1, ErrUnknownAddressDriver
	}

	return ty, nil
}
