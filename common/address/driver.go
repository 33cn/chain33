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
	drivers       = make(map[int32]*DriverInfo)
	defaultDriver Driver
	driverMutex   sync.Mutex
)

// Driver address driver
type Driver interface {

	// PubKeyToAddr public key to address
	PubKeyToAddr(pubKey []byte) string
	// ValidateAddr address validation
	ValidateAddr(addr string) error
	// GetName get driver name
	GetName(id int32) string
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
	IDMask   = 0x00007000
	IDOffset = 12
)

// DecodeAddressID, decode address id from signature type id
func DecodeAddressID(signID int32) int32 {
	return int32(IDMask) & signID >> IDOffset
}

// RegisterDriver 注册地址驱动
// enableHeight, 设置默认启用高度, 负数表示不启用
func RegisterDriver(id int32, driver Driver, enableHeight int64) {

	driverMutex.Lock()
	defer driverMutex.Unlock()
	if id < 0 || id > MaxID {
		panic(fmt.Sprintf("address id must in range [0, %d]", MaxID))
	}
	_, ok := drivers[id]
	if ok {
		panic(fmt.Sprintf("Register duplicate Address id %d", id))
	}
	info := &DriverInfo{
		driver:       driver,
		enableHeight: enableHeight,
	}
	drivers[id] = info
}

// LoadDriver 根据ID加载插件, 根据区块高度判定是否已启动
// 不关心启用状态, blockHeight传-1
func LoadDriver(id int32, blockHeight int64) (Driver, error) {

	base, ok := drivers[id]
	if !ok {
		return nil, ErrUnknownAddressDriver
	}

	if blockHeight >= 0 && base.enableHeight > blockHeight {
		return nil, ErrAddressDriverNotEnable
	}

	return base.driver, nil
}

// GetDriverList get driver list
func GetDriverList() []Driver {

	list := make([]Driver, 0, len(drivers))
	for _, d := range drivers {
		list = append(list, d.driver)
	}
	return list
}
