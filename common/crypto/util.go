// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package crypto

import (
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/tjfoc/gmsm/sm3"
	"golang.org/x/crypto/ripemd160"
)

//Sha256 加密算法
func Sha256(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

//Ripemd160 加密算法
func Ripemd160(bytes []byte) []byte {
	hasher := ripemd160.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

//Sm3Hash 加密算法
func Sm3Hash(msg []byte) []byte {
	c := sm3.New()
	c.Write(msg)
	return c.Sum(nil)
}

// BasicValidation 公私钥数据签名验证基础实现
func BasicValidation(c Crypto, msg, pub, sig []byte) error {

	pubKey, err := c.PubKeyFromBytes(pub)
	if err != nil {
		return err
	}
	s, err := c.SignatureFromBytes(sig)
	if err != nil {
		return err
	}
	if !pubKey.VerifyBytes(msg, s) {
		return ErrSign
	}
	return nil
}

//ToAggregate 判断签名是否可以支持聚合签名，并且返回聚合签名的接口
func ToAggregate(c Crypto) (AggregateCrypto, error) {
	if aggr, ok := c.(AggregateCrypto); ok {
		return aggr, nil
	}
	return nil, ErrNotSupportAggr
}

// WithOptionCGO 设置为CGO版本
func WithOptionCGO() Option {
	return func(d *Driver) error {
		d.isCGO = true
		return nil
	}
}

// WithOptionDefaultDisable 设置默认不启用
func WithOptionDefaultDisable() Option {
	return func(d *Driver) error {
		d.enableHeight = -1
		return nil
	}
}

// MaxManualTypeID 手动指定ID最大值 65534
const MaxManualTypeID = math.MaxUint16 - 1

// WithOptionTypeID 手动指定typeID， 不指定情况，系统将根据name自动生成typeID
func WithOptionTypeID(id int32) Option {
	return func(d *Driver) error {
		if id <= 0 {
			return errors.New("TypeIDMustPositive")
		}
		if id > MaxManualTypeID {
			return fmt.Errorf("TypeIDMustLessThan %d", MaxManualTypeID+1)
		}
		d.typeID = id
		return nil
	}
}

// WithOptionInitFunc 设置插件初始化接口
func WithOptionInitFunc(fn DriverInitFunc) Option {
	return func(d *Driver) error {
		if fn == nil {
			return errors.New("NilInitFunc")
		}
		d.initFunc = fn
		return nil
	}
}

// GenDriverTypeID 根据名称生成driver type id
func GenDriverTypeID(name string) int32 {
	buf := Sha256([]byte(name))
	id := int32(binary.BigEndian.Uint32(buf) % 1e8)
	// 自动生成的在区间[65535, 1e8+65535)
	return id + MaxManualTypeID + 1
}
