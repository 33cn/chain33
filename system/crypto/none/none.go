// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package none none driver
package none

import (
	"github.com/33cn/chain33/common/crypto"
)

// const
const (
	Name = "none"
	ID   = 10
)

func init() {
	// 默认启用高度-1， 不开启
	crypto.Register(Name, &Driver{}, crypto.WithRegOptionTypeID(ID), crypto.WithRegOptionDefaultDisable())
}

// Driver 驱动
type Driver struct{}

// GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	return nil, nil
}

// PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	return nil, nil
}

// PubKeyFromBytes 字节转为公钥
func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	return nil, nil
}

// SignatureFromBytes 字节转为签名
func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	return nil, nil
}

// Validate validate msg and signature
func (d Driver) Validate(msg, pub, sig []byte) error {
	return nil
}
