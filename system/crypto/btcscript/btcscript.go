// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package btcscript btc Script driver
package btcscript

import (
	"bytes"
	"errors"

	cryptocli "github.com/33cn/chain33/common/crypto/client"

	"github.com/33cn/chain33/system/crypto/btcscript/script"
	"github.com/33cn/chain33/system/crypto/secp256k1"

	"github.com/33cn/chain33/common"

	"github.com/33cn/chain33/common/crypto"
	"github.com/golang/protobuf/proto"
)

//const
const (
	Name = "btcscript"
	ID   = 11
)

func init() {
	// 默认启用高度-1， 不开启
	crypto.Register(Name, &Driver{}, crypto.WithRegOptionTypeID(ID))
}

//Driver 驱动, 除验证外，其余接口同secp256k1
type Driver struct {
	secp256k1.Driver
}

// Validate validate msg and signature
func (d Driver) Validate(msg, pk, sig []byte) error {
	ssig := &script.Signature{}
	err := proto.Unmarshal(sig, ssig)
	if err != nil {
		return err
	}

	//需要验证公钥和锁定脚本是否一一对应
	if !bytes.Equal(pk, common.Sha256(ssig.LockScript)) {
		return ErrInvalidLockScript
	}

	cryptocli.GetCryptoContext()

	// scriptLockTime <= lockTime <= blockTime
	// scriptSequence <= utxoSeq <= block delay time

	if err := script.CheckBtcScript(msg, ssig); err != nil {
		return ErrInvalidBtcSignature
	}
	return nil
}

var (

	// ErrInvalidLockScript invalid lock script
	ErrInvalidLockScript = errors.New("ErrInvalidLockScript")
	// ErrInvalidBtcSignature invalid btc signature
	ErrInvalidBtcSignature = errors.New("ErrInvalidBtcSignature")
)
