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
	//ErrAddressDriverNotEnable 驱动未启用
	ErrAddressDriverNotEnable = errors.New("ErrAddressDriverNotEnable")
)

var (
	drivers          = make(map[int32]*DriverInfo)
	driverName       = make(map[string]int32)
	defaultDriver    Driver
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
}

// DriverInfo driver info
type DriverInfo struct {
	driver       Driver
	enableHeight int64
}

const (
	// MaxID 最大id值
	MaxID = 7
	// IDMask
	IDMask = 0x00007000
	// offset
	IDOffset = 12
	// AnyID any valid id, not specified
	AnyID = MaxID + 1
	// DefaultID specify as default id
	DefaultID = -1
)

// Init init with config
func Init(config *Config) {

	driverMutex.Lock()
	defer driverMutex.Unlock()

	for name, enableHeight := range config.EnableHeight {

		id, ok := driverName[name]
		if !ok {
			panic(fmt.Sprintf("config address enable height, unkonwn driver \"%s\"", name))
		}
		drivers[id].enableHeight = enableHeight
	}

	defaultID, ok := driverName[config.DefaultDriver]
	if !ok {
		panic(fmt.Sprintf("config default driver, unknown driver \"%s\"", config.DefaultDriver))
	}

	info := drivers[defaultID]
	if info.enableHeight != 0 {
		panic(fmt.Sprintf("default driver \"%s\" enable height not equal 0", config.DefaultDriver))
	}
	defaultAddressID = defaultID
	defaultDriver = info.driver
}

// DecodeAddressID, decode address id from signature type id
func DecodeAddressID(signID int32) int32 {
	return int32(IDMask) & signID >> IDOffset
}

// EncodeAddressID, encode address id to sign id
func EncodeAddressID(signTy, addressID int32) int32 {
	if !isValidAddressID(addressID) {
		addressID = defaultAddressID
	}
	return addressID<<IDOffset | signTy
}

// RegisterDriver 注册地址驱动
// enableHeight, 设置默认启用高度, 负数表示不启用
func RegisterDriver(id int32, driver Driver, enableHeight int64) {

	driverMutex.Lock()
	defer driverMutex.Unlock()
	if !isValidAddressID(id) {
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

func isValidAddressID(id int32) bool {
	return id >= 0 && id <= MaxID
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
