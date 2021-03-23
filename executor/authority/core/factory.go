// Copyright Fuzamei Corp. 2018 All Rights Reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"errors"

	secp256r1_util "github.com/33cn/chain33/system/crypto/secp256r1"
	sm2_util "github.com/33cn/chain33/system/crypto/sm2"
)

// GetLocalValidator 根据类型获取校验器
func GetLocalValidator(authConfig *AuthConfig, signType int) (Validator, error) {
	var lclValidator Validator
	var err error

	if signType == secp256r1_util.ID {
		lclValidator = NewEcdsaValidator()
	} else if signType == sm2_util.ID {
		lclValidator = NewGmValidator()
	} else {
		return nil, errors.New("ErrUnknowAuthSignType")
	}

	err = lclValidator.Setup(authConfig)
	if err != nil {
		authLogger.Error("Failed to set up local validator config", "Error", err)
		return nil, errors.New("Failed to initialize local validator")
	}

	return lclValidator, nil
}
