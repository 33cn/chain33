// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package none none driver
package none

import (
	"github.com/33cn/chain33/common/crypto"
)

//const
const (
	Name = crypto.NameNone
	ID   = crypto.TyNone
)

func init() {
	crypto.Register(Name, &Driver{}, false)
	crypto.RegisterType(Name, ID)
}

//Driver 驱动
type Driver struct{}

//GenKey 生成私钥
func (d Driver) GenKey() (crypto.PrivKey, error) {
	return nil, nil
}

//PrivKeyFromBytes 字节转为私钥
func (d Driver) PrivKeyFromBytes(b []byte) (privKey crypto.PrivKey, err error) {
	return nil, nil
}

//PubKeyFromBytes 字节转为公钥
func (d Driver) PubKeyFromBytes(b []byte) (pubKey crypto.PubKey, err error) {
	return nil, nil
}

//SignatureFromBytes 字节转为签名
func (d Driver) SignatureFromBytes(b []byte) (sig crypto.Signature, err error) {
	return nil, nil
}

func (d Driver) Validate(msg, pub, sig []byte) error {
	return nil
}
