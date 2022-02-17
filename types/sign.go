// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
package types

import (
	"github.com/33cn/chain33/common/address"
	"github.com/33cn/chain33/common/crypto"
)

const (

	// CryptoIDMask crypto type id mask
	CryptoIDMask = 0xffff8fff
	// IDMask
	AddressIDMask = 0x00007000
	// AddressIDOffset
	AddressIDOffset = 12
)

// EncodeSignID encode sign id
func EncodeSignID(cryptoID, addressID int32) int32 {
	if !address.IsValidAddressID(addressID) {
		addressID = address.GetDefaultAddressID()
	}
	return (addressID << AddressIDOffset) | cryptoID
}

// ExtractAddressID extract address id from signature type id
func ExtractAddressID(signID int32) int32 {
	return int32(AddressIDMask) & signID >> AddressIDOffset
}

// ExtractCryptoID extract crypto id from signature type id
func ExtractCryptoID(signID int32) int32 {
	return int32(int(signID) & CryptoIDMask)
}

//GetSignName  获取签名类型
func GetSignName(execer string, signID int) string {
	//优先加载执行器的签名类型
	if execer != "" {
		exec := LoadExecutorType(execer)
		if exec != nil {
			name, err := exec.GetCryptoDriver(signID)
			if err == nil {
				return name
			}
		}
	}
	//加载系统执行器的签名类型
	return crypto.GetName(int(ExtractCryptoID(int32(signID))))
}

//GetSignType  获取签名类型
func GetSignType(execer string, name string) int {
	//优先加载执行器的签名类型
	if execer != "" {
		exec := LoadExecutorType(execer)
		if exec != nil {
			ty, err := exec.GetCryptoType(name)
			if err == nil {
				return ty
			}
		}
	}

	return int(EncodeSignID(int32(crypto.GetType(name)), address.GetDefaultAddressID()))
}
